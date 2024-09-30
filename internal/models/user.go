package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID             uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	FirstName      string    `gorm:"default:null" json:"first_name"`
	LastName       string    `gorm:"default:null" json:"last_name"`
	Email          string    `gorm:"unique;not null;default:null" json:"email"`
	HashedPassword string    `gorm:"default:null;check:(hashed_password IS NOT NULL OR is_external = true) " json:"-"`
	IsExternal     bool      `gorm:"not null;default:false" json:"is_external"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UserCreateBody struct {
	FirstName string `json:"first_name" validate:"omitempty"`
	LastName  string `json:"last_name" validate:"omitempty"`
	Email     string `json:"email" validate:"required,omitempty,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

type UserUpdateBody struct {
	FirstName string `json:"first_name" validate:"omitempty"`
	LastName  string `json:"last_name" validate:"omitempty"`
	Email     string `json:"email" validate:"omitempty,email"`
}
