package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileStatus string

const (
	FileStatusUploading FileStatus = "uploading"
	FileStatusUploaded  FileStatus = "uploaded"
	FileStatusDeleting  FileStatus = "deleting"
	FileStatusTrashed   FileStatus = "trashed"
	FileStatusRestoring FileStatus = "restoring"
)

type File struct {
	ID          uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"not null;default:null" json:"name"`
	Extension   string         `gorm:"default:null" json:"extension"`
	Status      FileStatus     `gorm:"type:file_status;default:null" json:"status"`
	BucketId    uuid.UUID      `gorm:"type:uuid;" json:"bucket_id"`
	Bucket      Bucket         `json:"-"`
	Path        string         `gorm:"not null;default:/" json:"path"`
	Type        string         `gorm:"not null;default:null" json:"type"`
	Size        int            `gorm:"default:null" json:"size"`
	TrashedAt   *time.Time     `gorm:"default:null;index" json:"trashed_at,omitempty"`
	TrashedBy   *uuid.UUID     `gorm:"type:uuid;default:null" json:"trashed_by,omitempty"`
	TrashedUser User           `gorm:"foreignKey:TrashedBy" json:"trashed_user,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type FileTransferBody struct {
	Name string `json:"name" validate:"required,filename,max=255"`
	Path string `json:"path" validate:"required,max=1024"`
	Type string `json:"type" validate:"required,oneof=file folder"`
	Size int    `json:"size" validate:"required_if=Type file,min=1,max=1099511627776"`
}

type FileTransferResponse struct {
	ID   string            `json:"id"`
	Url  string            `json:"url"`
	Body map[string]string `json:"body"`
}
