package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShareType string

const (
	ShareTypeFiles  ShareType = "files"
	ShareTypeFolder ShareType = "folder"
	ShareTypeBucket ShareType = "bucket"
)

type Share struct {
	ID                uuid.UUID      `gorm:"default:(-)"            json:"id"`
	Name              string         `gorm:"not null"               json:"name"`
	BucketID          uuid.UUID      `gorm:"not null"               json:"bucket_id"`
	Bucket            Bucket         `                              json:"-"`
	FolderID          *uuid.UUID     `gorm:"default:null"           json:"folder_id,omitempty"`
	Folder            *Folder        `gorm:"foreignKey:FolderID"    json:"-"`
	ExpiresAt         *time.Time     `gorm:"default:null"           json:"expires_at,omitempty"`
	MaxViews          *int           `gorm:"default:null"           json:"max_views,omitempty"`
	CurrentViews      int            `gorm:"not null;default:0"     json:"current_views"`
	HashedPassword    string         `gorm:"default:null"           json:"-"`
	PasswordProtected bool           `gorm:"-"                      json:"password_protected"`
	Type              ShareType      `gorm:"not null"               json:"type"`
	AllowUpload       bool           `gorm:"not null;default:false" json:"allow_upload"`
	MaxUploads        *int           `gorm:"default:null"           json:"max_uploads,omitempty"`
	CurrentUploads    int            `gorm:"not null;default:0"     json:"current_uploads"`
	MaxUploadSize     *int64         `gorm:"default:null"           json:"max_upload_size,omitempty"`
	Files             []ShareFile    `                              json:"files,omitempty"`
	CreatedBy         uuid.UUID      `gorm:"not null"               json:"created_by"`
	CreatedAt         time.Time      `                              json:"created_at"`
	UpdatedAt         time.Time      `                              json:"updated_at"`
	DeletedAt         gorm.DeletedAt `                              json:"deleted_at"`
}

type ShareActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type ShareType `json:"type"`
}

func (s *Share) ToActivity() ShareActivity {
	return ShareActivity{
		ID:   s.ID,
		Name: s.Name,
		Type: s.Type,
	}
}

type ShareFile struct {
	ID      uuid.UUID `gorm:"default:(-)" json:"id"`
	ShareID uuid.UUID `gorm:"not null"    json:"share_id"`
	FileID  uuid.UUID `gorm:"not null"    json:"file_id"`
	File    File      `                   json:"file"`
}

type PublicShareResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Type           ShareType  `json:"type"`
	AllowUpload    bool       `json:"allow_upload"`
	MaxUploadSize  *int64     `json:"max_upload_size"`
	MaxUploads     *int       `json:"max_uploads"`
	CurrentUploads int        `json:"current_uploads"`
	ExpiresAt      *time.Time `json:"expires_at"`
	MaxViews       *int       `json:"max_views"`
	CurrentViews   int        `json:"current_views"`
	Files          []File     `json:"files"`
	Folders        []Folder   `json:"folders"`
}

type ShareUploadBody struct {
	Name     string     `json:"name"      validate:"required,filename,max=255"`
	FolderID *uuid.UUID `json:"folder_id" validate:"omitempty,uuid"`
	Size     int64      `json:"size"      validate:"required,gte=1"`
}

type ShareAuthBody struct {
	Password string `json:"password" validate:"required,min=8"`
}

type ShareAuthResponse struct {
	Token string `json:"token"`
}

type ShareCreateBody struct {
	Name          string      `json:"name"            validate:"required,min=1,max=255"`
	Type          ShareType   `json:"type"            validate:"required,oneof=files folder bucket"`
	FolderID      *uuid.UUID  `json:"folder_id"       validate:"required_if=Type folder,omitempty,uuid"`
	FileIDs       []uuid.UUID `json:"file_ids"        validate:"required_if=Type files,omitempty,dive,uuid"`
	ExpiresAt     *time.Time  `json:"expires_at"      validate:"omitempty,futuredate"`
	MaxViews      *int        `json:"max_views"       validate:"omitempty,gte=1"`
	Password      string      `json:"password"        validate:"omitempty,min=8,max=72"`
	AllowUpload   bool        `json:"allow_upload"    validate:"excluded_if=Type files"`
	MaxUploads    *int        `json:"max_uploads"     validate:"omitempty,gte=1"`
	MaxUploadSize *int64      `json:"max_upload_size" validate:"omitempty,gte=1"`
}
