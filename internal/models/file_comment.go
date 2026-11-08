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

type FileCommentCreateBody struct {
	Body string `json:"body" validate:"required,max=2000"`
}

type FileCommentUpdateBody struct {
	Body string `json:"body" validate:"required,max=2000"`
}

type FileCommentCreateResponse struct {
	ID uuid.UUID `json:"id"`
}

type FileCommentResponse struct {
	ID         uuid.UUID `json:"id"`
	User       *UserInfo `json:"user,omitempty"`
	UserActive bool      `json:"user_active"`
	Body       string    `json:"body"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type FileConversationResponse struct {
	Messages []FileCommentResponse `json:"messages"`
}
