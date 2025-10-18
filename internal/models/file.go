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
)

type FileType string

const (
	FileTypeFile   FileType = "file"
	FileTypeFolder FileType = "folder"
)

type File struct {
	ID        uuid.UUID  `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name      string     `gorm:"not null;default:null" json:"name"`
	Extension string     `gorm:"default:null" json:"extension"`
	Status    FileStatus `gorm:"type:file_status;default:null" json:"status"`
	BucketID  uuid.UUID  `gorm:"type:uuid;" json:"bucket_id"`
	Bucket    Bucket     `json:"-"`
	Path      string     `gorm:"not null;default:/" json:"path"`
	Type      FileType   `gorm:"type:file_type;not null;default:null" json:"type"`
	Size      int        `gorm:"default:null;type:bigint" json:"size"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type FileTransferBody struct {
	Name string   `json:"name" validate:"required,filename"`
	Path string   `json:"path" validate:"required"`
	Type FileType `json:"type" validate:"required,oneof=file folder"`
	Size int      `json:"size" validate:"required_if=Type file"`
}

type FileTransferResponse struct {
	ID   string            `json:"id"`
	Url  string            `json:"url"`
	Body map[string]string `json:"body"`
}
