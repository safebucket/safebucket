package models

import (
	"time"

	"github.com/google/uuid"
)

type QueueOutboxMessage struct {
	ID           uuid.UUID `gorm:"default:(-)"`
	MessageType  string    `gorm:"not null"`
	Payload      string    `gorm:"not null"`
	Attempts     int       `gorm:"not null;default:0"`
	AvailableAt  time.Time `gorm:"not null"`
	LastError    *string
	ClaimID      *string
	ClaimedUntil *time.Time
	CreatedAt    time.Time
}

type ActivityOutboxMessage struct {
	ID           uuid.UUID `gorm:"default:(-)"`
	Payload      string    `gorm:"not null"`
	Attempts     int       `gorm:"not null;default:0"`
	AvailableAt  time.Time `gorm:"not null"`
	LastError    *string
	ClaimID      *string
	ClaimedUntil *time.Time
	CreatedAt    time.Time
}
