package models

import (
	"time"

	"github.com/google/uuid"
)

type FileComment struct {
	ID        uuid.UUID  `gorm:"default:(-)"`
	FileID    uuid.UUID  `gorm:"not null"`
	UserID    *uuid.UUID `gorm:"default:null"`
	User      *User
	Body      string `gorm:"not null"`
	Version   int    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
