package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	FirstName string `gorm:"not null;default:null" json:"first_name" validate:"required"`
	LastName  string `gorm:"not null;default:null" json:"last_name" validate:"required"`
	Email     string `gorm:"unique;not null;default:null" json:"email" validate:"required,email"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UserUpdateBody struct {
	FirstName string `json:"first_name" validate:"omitempty"`
	LastName  string `json:"last_name" validate:"omitempty"`
	Email     string `json:"email" validate:"omitempty,email"`
}
