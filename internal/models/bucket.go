package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Bucket struct {
	ID    uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name  string    `gorm:"not null;default:null" json:"name" validate:"required"`
	Files []File    `json:"files"`

	CreatedAt time.Time      `json:"created_at"`
	CreatedBy time.Time      `json:"created_by"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type ShareWith struct {
	Email string `json:"email" validate:"required,email"`
	Group string `json:"group" validate:"required,oneof=owner contributor viewer"`
}

type BucketCreateBody struct {
	Name      string      `json:"name" validate:"required"`
	ShareWith []ShareWith `json:"share_with" validate:"omitempty"`
}
