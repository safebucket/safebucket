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
	TrashedAt    *time.Time     `gorm:"default:null;index"                             json:"trashed_at,omitempty"`
	TrashedBy    *uuid.UUID     `gorm:"type:uuid;default:null"                         json:"trashed_by,omitempty"`
	TrashedUser  User           `gorm:"foreignKey:TrashedBy"                           json:"trashed_user,omitempty"`
	CreatedAt    time.Time      `                                                      json:"created_at"`
	UpdatedAt    time.Time      `                                                      json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"                                          json:"-"`
}

type FolderCreateBody struct {
	Name     string     `json:"name"      validate:"required,foldername,max=255"`
	FolderID *uuid.UUID `json:"folder_id" validate:"omitempty,uuid"`
}

type FolderUpdateBody struct {
	Name string `json:"name" validate:"required,foldername,max=255"`
}
