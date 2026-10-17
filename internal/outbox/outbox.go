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

type Event interface {
	EventType() string
	EventPayload() any
}

func AddEvents(tx *gorm.DB, events ...Event) error {
	if len(events) == 0 {
		return nil
	}

	now := time.Now().UTC()
	rows := make([]models.OutboxMessage, 0, len(events))

	for _, event := range events {
		eventType := event.EventType()
		if eventType == "" {
			return ErrEmptyEventType
		}

		payload, err := json.Marshal(event.EventPayload())
		if err != nil {
			return fmt.Errorf("failed to marshal %s payload: %w", eventType, err)
		}

		rows = append(rows, models.OutboxMessage{
			Kind:          models.OutboxMessageKindEvent,
			EventType:     &eventType,
			Payload:       string(payload),
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
	rows := make([]models.OutboxMessage, 0, len(activities))

	for _, item := range activities {
		payload, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("failed to marshal activity: %w", err)
		}

		rows = append(rows, models.OutboxMessage{
			Kind:          models.OutboxMessageKindActivity,
			Payload:       string(payload),
			NextAttemptAt: now,
			CreatedAt:     now,
		})
	}

	return tx.Create(&rows).Error
}
