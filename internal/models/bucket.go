package models

import (
	"gorm.io/gorm"
	"time"
)

type Bucket struct {
	ID   uint   `gorm:"primarykey" json:"id"`
	Name string `gorm:"not null;default:null" json:"name" validate:"required"`

	CreatedAt time.Time      `json:"created_at"`
	CreatedBy time.Time      `json:"created_by"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
