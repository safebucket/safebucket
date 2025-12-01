package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Folder struct {
	ID           uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name         string         `gorm:"not null;default:null"                          json:"name"`
	Status       FileStatus     `gorm:"type:file_status;default:null"                  json:"status"`
	FolderID     *uuid.UUID     `gorm:"type:uuid;default:null"                         json:"folder_id,omitempty"`
	ParentFolder *Folder        `gorm:"foreignKey:FolderID"                            json:"parent_folder,omitempty"`
	BucketID     uuid.UUID      `gorm:"type:uuid;not null"                             json:"bucket_id"`
	Bucket       Bucket         `                                                      json:"-"`
	TrashedBy    *uuid.UUID     `gorm:"type:uuid;default:null"                         json:"trashed_by,omitempty"`
	CreatedAt    time.Time      `                                                      json:"created_at"`
	UpdatedAt    time.Time      `                                                      json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"                                          json:"-"`
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
	Status FileStatus `json:"status" validate:"required,oneof=trashed uploaded"`
}
