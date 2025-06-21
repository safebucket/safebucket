package models

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID    uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Email string    `gorm:"not null;default:null;index:idx_invite_unique,unique" json:"email" validate:"required,email"`
	Group string    `gorm:"not null;default:null;index:idx_invite_unique" json:"group" validate:"required,oneof=owner contributor viewer"`

	BucketID uuid.UUID `gorm:"type:uuid;not null;index:idx_invite_unique" json:"bucket_id"`
	Bucket   Bucket    `gorm:"foreignKey:BucketID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"bucket"`

	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"-"`
	User      User      `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`

	CreatedAt time.Time `json:"created_at"`
}
