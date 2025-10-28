package models

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"                    json:"id"`
	Email     string    `gorm:"not null;default:null;index:idx_invite_unique,unique"              json:"email"      validate:"required,email"`
	Group     Group     `gorm:"type:group_type;not null;default:null;index:idx_invite_unique"     json:"group"      validate:"required,oneof=owner contributor viewer"`
	BucketID  uuid.UUID `gorm:"type:uuid;not null;index:idx_invite_unique"                        json:"bucket_id"`
	Bucket    Bucket    `gorm:"foreignKey:BucketID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"  json:"bucket"`
	CreatedBy uuid.UUID `gorm:"type:uuid;not null"                                                json:"-"`
	User      User      `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`
	CreatedAt time.Time `                                                                         json:"created_at"`
}

type InviteChallengeCreateBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

type InviteChallengeValidateBody struct {
	Code        string `json:"code"         validate:"required,len=6,alphanum"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}
