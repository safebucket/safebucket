package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	outboxBatchSize     = 50
	outboxPollInterval  = time.Second
	outboxRetryDelay    = 5 * time.Second
	outboxClaimDuration = time.Minute
)

type pendingOutboxMessage struct {
	ID          uuid.UUID
	MessageType string
	Payload     string
}

func processOutboxMessages(
	ctx context.Context,
	db *gorm.DB,
	tableName string,
	workerName string,
	deliver func(*pendingOutboxMessage) error,
) (int, error) {
	processed := 0
	var messages []pendingOutboxMessage
	now := time.Now()
	if err := db.WithContext(
		ctx,
	).Table(
		tableName,
	).Where(
		"available_at <= ? AND (claimed_until IS NULL OR claimed_until <= ?)", now, now,
	).Order(
		"created_at ASC",
	).Limit(
		outboxBatchSize,
	).Find(&messages).Error; err != nil {
		return 0, err
	}

	for i := range messages {
		if ctx.Err() != nil {
			return processed, nil
		}

		claimID := uuid.NewString()
		claimed, err := claimOutboxMessage(db, tableName, messages[i].ID, claimID)
		if err != nil {
			zap.L().Error("Failed to claim message",
				zap.String("worker", workerName),
				zap.String("outbox_id", messages[i].ID.String()),
				zap.Error(err))
			continue
		}
		if !claimed {
			continue
		}

		if deliveryErr := deliver(&messages[i]); deliveryErr != nil {
			if updateErr := db.Table(tableName).
				Where("id = ? AND claim_id = ?", messages[i].ID, claimID).
				Updates(map[string]any{
					"attempts":      gorm.Expr("attempts + 1"),
					"available_at":  time.Now().Add(outboxRetryDelay),
					"last_error":    deliveryErr.Error(),
					"claim_id":      nil,
					"claimed_until": nil,
				}).Error; updateErr != nil {
				zap.L().Error("Failed to record delivery failure",
					zap.String("worker", workerName),
					zap.String("outbox_id", messages[i].ID.String()),
					zap.Error(updateErr))
			}
			continue
		}

		result := db.Table(tableName).
			Where("id = ? AND claim_id = ?", messages[i].ID, claimID).
			Delete(&pendingOutboxMessage{})
		if result.Error != nil {
			zap.L().Error("Failed to delete delivered message",
				zap.String("worker", workerName),
				zap.String("outbox_id", messages[i].ID.String()),
				zap.Error(result.Error))
			continue
		}
		if result.RowsAffected == 1 {
			processed++
		}
	}

	return processed, nil
}

func claimOutboxMessage(db *gorm.DB, tableName string, id uuid.UUID, claimID string) (bool, error) {
	now := time.Now()
	result := db.Table(
		tableName,
	).Where(
		"id = ? AND available_at <= ? AND (claimed_until IS NULL OR claimed_until <= ?)", id, now, now,
	).Updates(map[string]any{
		"claim_id":      claimID,
		"claimed_until": now.Add(outboxClaimDuration),
	})

	return result.RowsAffected == 1, result.Error
}
