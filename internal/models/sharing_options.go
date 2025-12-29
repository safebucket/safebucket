package models

import (
	"time"

	"github.com/google/uuid"
)

type SharingOptions struct {
	ID            uuid.UUID  `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	FileID        uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"                 json:"file_id"`
	ExpiresAt     *time.Time `gorm:"default:null"                                   json:"expires_at,omitempty"`
	MaxDownloads  *int       `gorm:"default:null"                                   json:"max_downloads,omitempty"`
	DownloadCount int        `gorm:"not null;default:0"                             json:"download_count"`
	CreatedAt     time.Time  `                                                      json:"created_at"`
	UpdatedAt     time.Time  `                                                      json:"updated_at"`
}

// SharingOptionsBody is the request body for creating sharing options during file upload.
type SharingOptionsBody struct {
	ExpiresAt    *time.Time `json:"expires_at"    validate:"omitempty"`
	MaxDownloads *int       `json:"max_downloads" validate:"omitempty,min=1,max=10000"`
}

// HasOptions checks if any sharing options are set.
func (s *SharingOptionsBody) HasOptions() bool {
	return s != nil && (s.ExpiresAt != nil || s.MaxDownloads != nil)
}
