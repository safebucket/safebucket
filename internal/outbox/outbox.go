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

func AddEvent(tx *gorm.DB, name string, payload any) error {
	if name == "" {
		return ErrEmptyEventType
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %s payload: %w", name, err)
	}

	now := time.Now().UTC()

	return tx.Create(&models.Outbox{
		Type:          models.OutboxTypeEvent,
		Name:          &name,
		Payload:       string(body),
		NextAttemptAt: now,
		CreatedAt:     now,
	}).Error
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
