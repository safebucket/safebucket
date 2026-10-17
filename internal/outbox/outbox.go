package outbox

import (
	"encoding/json"
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"gorm.io/gorm"
)

func EnqueueEvent(tx *gorm.DB, eventType string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return tx.Create(&models.QueueOutboxMessage{
		MessageType: eventType,
		Payload:     string(encoded),
		AvailableAt: time.Now(),
	}).Error
}

func EnqueueActivity(tx *gorm.DB, activity models.Activity) error {
	encoded, err := json.Marshal(activity)
	if err != nil {
		return err
	}

	return tx.Create(&models.ActivityOutboxMessage{
		Payload:     string(encoded),
		AvailableAt: time.Now(),
	}).Error
}
