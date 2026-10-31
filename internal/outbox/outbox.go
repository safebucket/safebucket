package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"gorm.io/gorm"
)

var ErrEmptyEventType = errors.New("outbox event type is empty")

func AddEvents[T any](tx *gorm.DB, name string, payloads ...T) error {
	if name == "" {
		return ErrEmptyEventType
	}
	if len(payloads) == 0 {
		return nil
	}

	now := time.Now().UTC()
	rows := make([]models.Outbox, 0, len(payloads))

	for _, payload := range payloads {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal %s payload: %w", name, err)
		}

		rows = append(rows, models.Outbox{
			Type:          models.OutboxTypeEvent,
			Name:          &name,
			Payload:       string(body),
			NextAttemptAt: now,
			CreatedAt:     now,
		})
	}

	return tx.Create(&rows).Error
}

func AddActivities(tx *gorm.DB, activities ...models.Activity) error {
	if len(activities) == 0 {
		return nil
	}

	now := time.Now().UTC()
	rows := make([]models.Outbox, 0, len(activities))

	for _, item := range activities {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal activity: %w", err)
		}

		rows = append(rows, models.Outbox{
			Type:          models.OutboxTypeActivity,
			Payload:       string(payload),
			NextAttemptAt: now,
			CreatedAt:     now,
		})
	}

	return tx.Create(&rows).Error
}
