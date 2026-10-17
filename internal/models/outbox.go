package models

import (
	"time"

	"github.com/google/uuid"
)

type OutboxMessageKind string

const (
	OutboxMessageKindEvent    OutboxMessageKind = "event"
	OutboxMessageKindActivity OutboxMessageKind = "activity"
)

type OutboxMessage struct {
	ID            uuid.UUID         `gorm:"default:(-)"        json:"id"`
	Kind          OutboxMessageKind `gorm:"not null"           json:"kind"`
	EventType     *string           `gorm:"default:null"       json:"event_type,omitempty"`
	Payload       string            `gorm:"not null"           json:"payload"`
	Attempts      int               `gorm:"not null;default:0" json:"attempts"`
	LastError     *string           `gorm:"default:null"       json:"last_error,omitempty"`
	NextAttemptAt time.Time         `gorm:"not null"           json:"next_attempt_at"`
	FailedAt      *time.Time        `gorm:"default:null"       json:"failed_at,omitempty"`
	CreatedAt     time.Time         `                          json:"created_at"`
}
