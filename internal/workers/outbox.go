package workers

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	outboxMaxBackoff         = 5 * time.Minute
	outboxErrorLevelAttempts = 10
	outboxLastErrorMaxLength = 500
)

var errMissingOutboxEventType = errors.New("outbox event row has no event type")

type OutboxWorker struct {
	DB             *gorm.DB
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	PollInterval   time.Duration
	BatchSize      int
}

func (w *OutboxWorker) Start(ctx context.Context) {
	zap.L().Info("Starting worker", zap.String("worker", configuration.WorkerOutbox))

	var loops sync.WaitGroup
	loops.Go(func() { w.runLoop(ctx, models.OutboxMessageKindEvent, w.deliverEventBatch) })
	loops.Go(func() { w.runLoop(ctx, models.OutboxMessageKindActivity, w.deliverActivityBatch) })
	loops.Wait()
}

func (w *OutboxWorker) runLoop(
	ctx context.Context,
	kind models.OutboxMessageKind,
	deliverBatch func(context.Context) (int, error),
) {
	for {
		if ctx.Err() != nil {
			return
		}

		processed, err := deliverBatch(ctx)
		if err != nil {
			zap.L().Error("Outbox delivery failed",
				zap.String("kind", string(kind)),
				zap.Error(err))
		}

		if w.BatchSize > 0 && processed >= w.BatchSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(w.PollInterval):
		}
	}
}

func (w *OutboxWorker) deliverEventBatch(ctx context.Context) (int, error) {
	rows, err := w.pendingMessages(models.OutboxMessageKindEvent)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, row := range rows {
		if ctx.Err() != nil {
			break
		}

		if publishErr := w.publishEvent(row); publishErr != nil {
			if errors.Is(publishErr, errMissingOutboxEventType) {
				w.markMessagesFailed([]uuid.UUID{row.ID}, publishErr)
				processed++
				continue
			}

			w.recordFailure(row, publishErr)
			return processed, publishErr
		}

		if delErr := w.deleteMessages([]uuid.UUID{row.ID}); delErr != nil {
			return processed, delErr
		}
		processed++
	}

	if len(rows) > 0 {
		zap.L().Debug("Delivered outbox event batch",
			zap.Int("batch", len(rows)),
			zap.Int("processed", processed))
	}

	return processed, nil
}

func (w *OutboxWorker) publishEvent(row models.OutboxMessage) error {
	if row.EventType == nil || *row.EventType == "" {
		return errMissingOutboxEventType
	}

	msg := message.NewMessage(row.ID.String(), []byte(row.Payload))
	msg.Metadata.Set("type", *row.EventType)

	return w.Publisher.Publish(msg)
}

func (w *OutboxWorker) deliverActivityBatch(ctx context.Context) (int, error) {
	rows, err := w.pendingMessages(models.OutboxMessageKindActivity)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}

	activities := make([]models.Activity, 0, len(rows))
	deliveredRows := make([]models.OutboxMessage, 0, len(rows))
	processed := 0

	for _, row := range rows {
		if ctx.Err() != nil {
			break
		}

		var entry models.Activity
		if unmarshalErr := json.Unmarshal([]byte(row.Payload), &entry); unmarshalErr != nil {
			w.markMessagesFailed([]uuid.UUID{row.ID}, unmarshalErr)
			processed++
			continue
		}

		entry.ID = row.ID
		activities = append(activities, entry)
		deliveredRows = append(deliveredRows, row)
	}

	if len(activities) == 0 {
		return processed, nil
	}

	if sendErr := w.ActivityLogger.Send(activities...); sendErr != nil {
		if errors.Is(sendErr, activity.ErrRejected) {
			w.markMessagesFailed(outboxRowIDs(deliveredRows), sendErr)
			return processed + len(deliveredRows), nil
		}

		w.recordBatchFailure(deliveredRows, sendErr)
		return processed, sendErr
	}

	if delErr := w.deleteMessages(outboxRowIDs(deliveredRows)); delErr != nil {
		return processed, delErr
	}

	zap.L().Debug("Delivered outbox activity batch", zap.Int("batch", len(deliveredRows)))

	return processed + len(deliveredRows), nil
}

func (w *OutboxWorker) pendingMessages(kind models.OutboxMessageKind) ([]models.OutboxMessage, error) {
	var rows []models.OutboxMessage

	err := w.DB.
		Where("kind = ? AND failed_at IS NULL AND next_attempt_at <= ?", kind, time.Now().UTC()).
		Order("created_at").
		Limit(w.BatchSize).
		Find(&rows).Error

	return rows, err
}

func (w *OutboxWorker) deleteMessages(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	return w.DB.Where("id IN ?", ids).Delete(&models.OutboxMessage{}).Error
}

func (w *OutboxWorker) markMessagesFailed(ids []uuid.UUID, failure error) {
	if len(ids) == 0 {
		return
	}

	err := w.DB.Model(&models.OutboxMessage{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"failed_at":  time.Now().UTC(),
			"last_error": truncateOutboxError(failure.Error()),
		}).Error
	if err != nil {
		zap.L().Error("Failed to mark outbox messages as failed",
			zap.Int("count", len(ids)),
			zap.Error(err))
		return
	}

	zap.L().Error("Outbox messages rejected permanently",
		zap.Int("count", len(ids)),
		zap.Error(failure))
}

func (w *OutboxWorker) recordBatchFailure(rows []models.OutboxMessage, failure error) {
	for _, row := range rows {
		w.recordFailure(row, failure)
	}
}

func (w *OutboxWorker) recordFailure(row models.OutboxMessage, failure error) {
	attempts := row.Attempts + 1

	err := w.DB.Model(&models.OutboxMessage{}).
		Where("id = ?", row.ID).
		Updates(map[string]any{
			"attempts":        attempts,
			"last_error":      truncateOutboxError(failure.Error()),
			"next_attempt_at": time.Now().UTC().Add(outboxBackoff(attempts)),
		}).Error
	if err != nil {
		zap.L().Error("Failed to record outbox delivery failure",
			zap.String("message_id", row.ID.String()),
			zap.Error(err))
		return
	}

	if attempts >= outboxErrorLevelAttempts {
		zap.L().Error("Outbox delivery keeps failing",
			zap.String("message_id", row.ID.String()),
			zap.String("kind", string(row.Kind)),
			zap.Int("attempts", attempts),
			zap.Error(failure))
	}
}

func outboxRowIDs(rows []models.OutboxMessage) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}

	return ids
}

func outboxBackoff(attempts int) time.Duration {
	backoff := time.Second << min(attempts, 9)
	if backoff > outboxMaxBackoff {
		return outboxMaxBackoff
	}

	return backoff
}

func truncateOutboxError(message string) string {
	if len(message) <= outboxLastErrorMaxLength {
		return message
	}

	return message[:outboxLastErrorMaxLength]
}
