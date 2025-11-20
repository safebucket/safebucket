package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bucket struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"not null;default:null"                          json:"name"       validate:"required"`
	Files     []File         `                                                      json:"files"`
	Folders   []Folder       `                                                      json:"folders"`
	CreatedAt time.Time      `                                                      json:"created_at"`
	CreatedBy uuid.UUID      `gorm:"type:uuid;not null"                             json:"-"`
	UpdatedAt time.Time      `                                                      json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                          json:"-"`
}

type BucketCreateUpdateBody struct {
	Name string `json:"name" validate:"required,max=100"`
}
