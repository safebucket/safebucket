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

type BucketMember struct {
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Email     string    `json:"email" validate:"required"`
	FirstName string    `json:"first_name,omitempty"`
	LastName  string    `json:"last_name,omitempty"`
	Role      string    `json:"role" validate:"required,oneof=owner contributor viewer"`
	Status    string    `json:"status" validate:"required,oneof=active invited"`
}
