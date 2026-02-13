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
	FileStatusDeleted   FileStatus = "deleted"
	FileStatusRestoring FileStatus = "restoring"
)

type File struct {
	ID           uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name         string         `gorm:"not null;default:null"                          json:"name"`
	Extension    string         `gorm:"default:null"                                   json:"extension"`
	Status       FileStatus     `gorm:"type:file_status;default:null"                  json:"status"`
	BucketID     uuid.UUID      `gorm:"type:uuid;"                                     json:"bucket_id"`
	Bucket       Bucket         `                                                      json:"-"`
	FolderID     *uuid.UUID     `gorm:"type:uuid;default:null"                         json:"folder_id,omitempty"`
	ParentFolder *Folder        `gorm:"foreignKey:FolderID"                            json:"parent_folder,omitempty"`
	Size         int            `gorm:"type:bigint;default:null"                       json:"size"`
	DeletedBy    *uuid.UUID     `gorm:"type:uuid;default:null"                         json:"deleted_by,omitempty"`
	ExpiresAt    *time.Time     `gorm:"default:null"                                   json:"expires_at"`
	OriginalPath string         `gorm:"-"                                              json:"original_path,omitempty"`
	CreatedAt    time.Time      `                                                      json:"created_at"`
	UpdatedAt    time.Time      `                                                      json:"updated_at"`
	DeletedAt    gorm.DeletedAt `                                                      json:"deleted_at"`
}

type FileActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (f *File) ToActivity() FileActivity {
	return FileActivity{
		ID:   f.ID,
		Name: f.Name,
	}
}

type FileTransferBody struct {
	Name      string     `json:"name"                 validate:"required,filename,max=255"`
	FolderID  *uuid.UUID `json:"folder_id"            validate:"omitempty,uuid"`
	Size      int        `json:"size"                 validate:"required,gte=1,maxuploadsize"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" validate:"omitempty,futuredate"`
}

type FileTransferResponse struct {
	ID   string            `json:"id"`
	URL  string            `json:"url"`
	Body map[string]string `json:"body"`
}

// FilePatchBody represents a PATCH request for trash/restore operations.
type FilePatchBody struct {
	Status string `json:"status" validate:"required,oneof=deleted uploaded"`
}
