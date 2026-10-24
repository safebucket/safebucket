package workers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	outboxMaxBackoff         = 5 * time.Minute
	outboxLeaseDuration      = time.Minute
	outboxErrorLevelAttempts = 10
	outboxLastErrorMaxLength = 500
)

var (
	errMissingOutboxEventType = errors.New("outbox event row has no event type")
	errInvalidOutboxPayload   = errors.New("outbox row payload is invalid")
	errUnknownOutboxType      = errors.New("outbox row has unknown type")
)

type OutboxWorker struct {
	DB             *gorm.DB
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	PollInterval   time.Duration
	BatchSize      int
}

func (w *OutboxWorker) Start(ctx context.Context) {
	zap.L().Info("Starting worker", zap.String("worker", configuration.WorkerOutbox))
	for {
		if ctx.Err() != nil {
			return
		}

		processed, err := w.deliverBatch()
		if err != nil {
			zap.L().Error("Outbox delivery failed", zap.Error(err))
		}

		if err == nil && w.BatchSize > 0 && processed >= w.BatchSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(w.PollInterval):
		}
	}
}

func (w *OutboxWorker) deliverBatch() (int, error) {
	rows, err := w.claimBatch()
	if err != nil {
		return 0, err
	}

	if len(rows) == 0 {
		return 0, nil
	}

	events := make([]models.Outbox, 0, len(rows))
	activities := make([]models.Outbox, 0, len(rows))

	for _, row := range rows {
		switch row.Type {
		case models.OutboxTypeEvent:
			events = append(events, row)
		case models.OutboxTypeActivity:
			activities = append(activities, row)
		default:
			w.markMessagesFailed([]uuid.UUID{row.ID}, fmt.Errorf("%w: %q", errUnknownOutboxType, row.Type))
		}
	}

	return len(rows), errors.Join(w.deliverEvents(events), w.deliverActivities(activities))
}

func (w *OutboxWorker) claimBatch() ([]models.Outbox, error) {
	var rows []models.Outbox

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("failed_at IS NULL AND next_attempt_at <= ?", time.Now().UTC()).
			Order("created_at").
			Limit(w.BatchSize).
			Find(&rows).Error

		if err != nil || len(rows) == 0 {
			return err
		}

		return tx.Model(&models.Outbox{}).
			Where("id IN ?", outboxRowIDs(rows)).
			Update("next_attempt_at", time.Now().UTC().Add(outboxLeaseDuration)).Error
	})

	return rows, err
}

func (w *OutboxWorker) deliverEvents(rows []models.Outbox) error {
	messages := make([]*message.Message, 0, len(rows))
	deliveredRows := make([]models.Outbox, 0, len(rows))

	invalidRows := make([]uuid.UUID, 0)

	for _, row := range rows {
		if row.Name == nil || *row.Name == "" {
			invalidRows = append(invalidRows, row.ID)
			continue
		}

		msg := message.NewMessage(row.ID.String(), []byte(row.Payload))
		msg.Metadata.Set("type", *row.Name)
		messages = append(messages, msg)
		deliveredRows = append(deliveredRows, row)
	}

	w.markMessagesFailed(invalidRows, errMissingOutboxEventType)

	if len(messages) == 0 {
		return nil
	}

	if err := w.Publisher.Publish(messages...); err != nil {
		w.recordFailures(deliveredRows, err)

		return err
	}

	w.deleteMessages(deliveredRows...)

	return nil
}

func (w *OutboxWorker) deliverActivities(rows []models.Outbox) error {
	activities := make([]models.Activity, 0, len(rows))
	deliveredRows := make([]models.Outbox, 0, len(rows))

	for _, row := range rows {
		var entry models.Activity
		if err := json.Unmarshal([]byte(row.Payload), &entry); err != nil {
			w.markMessagesFailed([]uuid.UUID{row.ID}, fmt.Errorf("%w: %w", errInvalidOutboxPayload, err))
			continue
		}

		entry.ID = row.ID
		activities = append(activities, entry)
		deliveredRows = append(deliveredRows, row)
	}

	if len(activities) == 0 {
		return nil
	}

	if err := w.ActivityLogger.Send(activities...); err != nil {
		if errors.Is(err, activity.ErrRejected) {
			w.markMessagesFailed(outboxRowIDs(deliveredRows), err)

			return nil
		}

		w.recordFailures(deliveredRows, err)

		return err
	}

	w.deleteMessages(deliveredRows...)

	return nil
}

func (w *OutboxWorker) deleteMessages(rows ...models.Outbox) {
	if len(rows) == 0 {
		return
	}

	ids := outboxRowIDs(rows)

	err := w.DB.Where("id IN ?", ids).Delete(&models.Outbox{}).Error
	if err != nil {
		zap.L().Error("Failed to delete delivered outbox messages",
			zap.Int("count", len(ids)),
			zap.Error(err))
	}
}

func (w *OutboxWorker) markMessagesFailed(ids []uuid.UUID, failure error) {
	if len(ids) == 0 {
		return
	}

	err := w.DB.Model(&models.Outbox{}).
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

func (w *OutboxWorker) recordFailures(rows []models.Outbox, failure error) {
	if len(rows) == 0 {
		return
	}

	lastError := truncateOutboxError(failure.Error())
	now := time.Now().UTC()

	for _, row := range rows {
		attempts := row.Attempts + 1

		err := w.DB.Model(&models.Outbox{}).
			Where("id = ?", row.ID).
			Updates(map[string]any{
				"attempts":        attempts,
				"last_error":      lastError,
				"next_attempt_at": now.Add(outboxBackoff(attempts)),
			}).Error
		if err != nil {
			zap.L().Error("Failed to record outbox delivery failure",
				zap.String("message_id", row.ID.String()),
				zap.Error(err))
			continue
		}

		if attempts >= outboxErrorLevelAttempts {
			zap.L().Error("Outbox delivery keeps failing",
				zap.String("message_id", row.ID.String()),
				zap.String("kind", string(row.Type)),
				zap.Int("attempts", attempts),
				zap.Error(failure))
		}
	}
}

func outboxRowIDs(rows []models.Outbox) []uuid.UUID {
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
