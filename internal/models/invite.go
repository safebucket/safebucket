package models

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"not null;default:null;index:idx_invite_unique,unique" json:"email" validate:"required,email"`
	Group     string    `gorm:"not null;default:null;index:idx_invite_unique" json:"group" validate:"required,oneof=owner contributor viewer"`
	BucketID  uuid.UUID `gorm:"type:uuid;not null;index:idx_invite_unique" json:"bucket_id"`
	Bucket    Bucket    `gorm:"foreignKey:BucketID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"bucket"`
	CreatedBy uuid.UUID `gorm:"type:uuid;not null" json:"-"`
	User      User      `gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`
	CreatedAt time.Time `json:"created_at"`
}

type InviteChallengeCreateBody struct {
	Email string `json:"email" validate:"required,email"`
}

type Challenge struct {
	ID           uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	InviteID     uuid.UUID `gorm:"type:uuid;not null;index:idx_challenge_unique,unique" json:"invite_id"`
	Invite       Invite    `gorm:"foreignKey:InviteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"invite"`
	HashedSecret string    `gorm:"not null;default:null" json:"hashed_secret" validate:"required"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type InviteChallengeValidateBody struct {
	Code string `json:"code" validate:"required"` // TODO: validate value
}

type BucketInvitee struct {
	Email string `json:"email" validate:"required,email"`
	Group string `json:"group" validate:"required,oneof=owner contributor viewer"`
}

type InviteBody struct {
	BucketID uuid.UUID       `json:"bucket_id" validate:"required,uuid"`
	Invites  []BucketInvitee `json:"invites" validate:"required,dive"`
}

type InviteResult struct {
	Email  string `json:"email"`
	Group  string `json:"group"`
	Status string `json:"status"`
}
