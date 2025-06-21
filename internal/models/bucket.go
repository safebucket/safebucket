package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bucket struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"not null;default:null" json:"name" validate:"required"`
	Files     []File         `json:"files"`
	CreatedAt time.Time      `json:"created_at"`
	CreatedBy uuid.UUID      `gorm:"type:uuid;not null" json:"-"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type BucketCreateBody struct {
	Name string `json:"name" validate:"required"`
}

type BucketInvitee struct {
	Email string `json:"email" validate:"required,email"`
	Group string `json:"group" validate:"required,oneof=owner contributor viewer"`
}

type BucketInviteBody struct {
	Invites []BucketInvitee `json:"invites" validate:"required,dive"`
}

type BucketInviteResult struct {
	Email  string `json:"email"`
	Group  string `json:"group"`
	Status string `json:"status"`
}
