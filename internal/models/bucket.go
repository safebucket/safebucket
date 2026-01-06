package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bucket struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Name      string         `gorm:"not null;default:null"                          json:"name"       validate:"required"`
	Files     []File         `                                                      json:"files"`
	Folders   []Folder       `                                                      json:"folders"`
	CreatedAt time.Time      `                                                      json:"created_at"`
	CreatedBy uuid.UUID      `gorm:"type:uuid;not null"                             json:"-"`
	UpdatedAt time.Time      `                                                      json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                          json:"deleted_at"`
}

type BucketActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (b *Bucket) ToActivity() BucketActivity {
	return BucketActivity{
		ID:   b.ID,
		Name: b.Name,
	}
}

type BucketCreateUpdateBody struct {
	Name string `json:"name" validate:"required,max=100"`
}

// AdminBucketListItem represents a bucket with enriched admin information.
type AdminBucketListItem struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Creator     UserActivity `json:"creator"`
	MemberCount int64        `json:"member_count"`
	FileCount   int64        `json:"file_count"`
	Size        int64        `json:"size"`
}

// BucketQueryParams defines query parameters for filtering bucket contents.
// Use with the ValidateQuery middleware:
//
//	r.With(m.ValidateQuery[models.BucketQueryParams]).
//	    Get("/", handler)
type BucketQueryParams struct {
	Status string `json:"status" validate:"omitempty,oneof=all deleted uploaded uploading"`
}
