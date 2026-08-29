package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FolderStatus string

const (
	FolderStatusCreated   FolderStatus = "created"
	FolderStatusDeleted   FolderStatus = "deleted"
	FolderStatusRestoring FolderStatus = "restoring"
)

type Folder struct {
	ID           uuid.UUID      `gorm:"default:(-)"              json:"id"`
	Name         string         `gorm:"not null;default:null"    json:"name"`
	Status       FolderStatus   `gorm:"not null;default:created" json:"status"`
	FolderID     *uuid.UUID     `gorm:"default:null"             json:"folder_id,omitempty"`
	ParentFolder *Folder        `gorm:"foreignKey:FolderID"      json:"parent_folder,omitempty"`
	BucketID     uuid.UUID      `gorm:"not null"                 json:"bucket_id"`
	Bucket       Bucket         `                                json:"-"`
	DeletedBy    *uuid.UUID     `gorm:"default:null"             json:"deleted_by,omitempty"`
	OriginalPath string         `gorm:"-"                        json:"original_path,omitempty"`
	CreatedAt    time.Time      `                                json:"created_at"`
	UpdatedAt    time.Time      `                                json:"updated_at"`
	DeletedAt    gorm.DeletedAt `                                json:"deleted_at"`
}

type FolderActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (f *Folder) ToActivity() FolderActivity {
	return FolderActivity{
		ID:   f.ID,
		Name: f.Name,
	}
}

type FolderCreateBody struct {
	Name     string     `json:"name"      validate:"required,foldername,max=255"`
	FolderID *uuid.UUID `json:"folder_id" validate:"omitempty,uuid"`
}

type FolderUpdateBody struct {
	Name string `json:"name" validate:"required,foldername,max=255"`
}

type FolderPatchBody struct {
	Status FolderStatus `json:"status" validate:"required,oneof=deleted created"`
}
