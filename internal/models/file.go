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
	FileStatusDeleted   FileStatus = "deleted"
	FileStatusRestoring FileStatus = "restoring"
)

type File struct {
	ID               uuid.UUID      `gorm:"default:(-)"           json:"id"`
	Name             string         `gorm:"not null;default:null" json:"name"`
	Extension        string         `gorm:"default:null"          json:"extension"`
	Status           FileStatus     `gorm:"default:null"          json:"status"`
	BucketID         uuid.UUID      `                             json:"bucket_id"`
	Bucket           Bucket         `                             json:"-"`
	FolderID         *uuid.UUID     `gorm:"default:null"          json:"folder_id,omitempty"`
	ParentFolder     *Folder        `gorm:"foreignKey:FolderID"   json:"parent_folder,omitempty"`
	Size             int            `gorm:"not null;default:0"    json:"size"`
	CurrentVersionID *uuid.UUID     `gorm:"default:null"          json:"current_version_id,omitempty"`
	DeletedBy        *uuid.UUID     `gorm:"default:null"          json:"deleted_by,omitempty"`
	ExpiresAt        *time.Time     `gorm:"default:null"          json:"expires_at"`
	OriginalPath     string         `gorm:"-"                     json:"original_path,omitempty"`
	CreatedAt        time.Time      `                             json:"created_at"`
	UpdatedAt        time.Time      `                             json:"updated_at"`
	DeletedAt        gorm.DeletedAt `                             json:"deleted_at"`
}

type FileVersion struct {
	ID         uuid.UUID  `gorm:"default:(-)"        json:"id"`
	FileID     uuid.UUID  `                          json:"file_id"`
	Version    int        `                          json:"version"`
	Size       int        `gorm:"not null;default:0" json:"size"`
	Status     FileStatus `gorm:"default:null"       json:"status"`
	UploadedBy *uuid.UUID `gorm:"default:null"       json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `                          json:"created_at"`
}

type FileVersionResponse struct {
	ID         uuid.UUID  `json:"id"`
	Version    int        `json:"version"`
	Size       int        `json:"size"`
	Status     FileStatus `json:"status"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty"`
	IsCurrent  bool       `json:"is_current"`
	CreatedAt  time.Time  `json:"created_at"`
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

type FileUploadBody struct {
	Name      string     `json:"name"                 validate:"required,filename,max=255"`
	FolderID  *uuid.UUID `json:"folder_id"            validate:"omitempty,uuid"`
	Size      int        `json:"size"                 validate:"required,gte=1,maxuploadsize"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" validate:"omitempty,futuredate"`
}

type FileUploadResponse struct {
	ID     string              `json:"id"`
	Method string              `json:"method"`
	URL    string              `json:"url,omitempty"`
	Body   []map[string]string `json:"body,omitempty"`
	Parts  []FilePartURL       `json:"parts,omitempty"`
}

type FilePartURL struct {
	ID      int               `json:"id"`
	URL     string            `json:"url"`
	Size    int64             `json:"size"`
	Headers map[string]string `json:"headers,omitempty"`
}

type FileDownloadResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type FileDownloadQuery struct {
	Context string `json:"context" validate:"omitempty,oneof=preview download"`
}

type FilePatchBody struct {
	Status string `json:"status" validate:"required,oneof=deleted uploaded"`
}

type FileVersionRestoreBody struct {
	VersionID uuid.UUID `json:"version_id" validate:"required"`
}
