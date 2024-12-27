package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type File struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"not null;default:null" json:"name"`
	Extension string    `gorm:"default:null" json:"extension"`
	Uploaded  bool      `gorm:"not null;default:false" json:"uploaded"`
	BucketId  uuid.UUID `gorm:"type:uuid;" json:"bucket_id"`
	Bucket    Bucket    `json:"-"`
	Path      string    `gorm:"not null;default:/" json:"path"`
	Type      string    `gorm:"not null;default:null" json:"type"`
	Size      int       `gorm:"default:null" json:"size"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type FileTransferBody struct {
	Name string `json:"name" validate:"required,filename"`
	Path string `json:"path" validate:"required"`
	Type string `json:"type" validate:"required,oneof=file folder"`
	Size int    `json:"size" validate:"required_if=Type file"`
}

type FileTransferResponse struct {
	ID   string            `json:"id"`
	Url  string            `json:"url"`
	Body map[string]string `json:"body"`
}

type UpdateFileBody struct {
	Uploaded *bool `json:"uploaded" validate:"omitempty,required_with=Uploaded,eq=true"`
}
