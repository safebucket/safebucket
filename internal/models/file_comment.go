package models

import (
	"time"

	"github.com/google/uuid"
)

type FileComment struct {
	ID        uuid.UUID  `gorm:"default:(-)"`
	FileID    uuid.UUID  `gorm:"not null"`
	AuthorID  *uuid.UUID `gorm:"default:null"`
	Author    *User      `gorm:"foreignKey:AuthorID"`
	Body      string     `gorm:"not null"`
	Version   int        `gorm:"not null"`
	EditedAt  *time.Time `gorm:"default:null"`
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
	ID           uuid.UUID  `json:"id"`
	Author       *UserInfo  `json:"author,omitempty"`
	AuthorActive bool       `json:"author_active"`
	Body         string     `json:"body"`
	Version      int        `json:"version"`
	EditedAt     *time.Time `json:"edited_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type FileConversationResponse struct {
	Messages []FileCommentResponse `json:"messages"`
}
