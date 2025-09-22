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
	CreatedAt    time.Time `json:"created_at"`
}

type InviteChallengeValidateBody struct {
	Code string `json:"code" validate:"required"` // TODO: validate value
}

type PasswordResetRequestBody struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordResetChallenge struct {
	ID           uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index:idx_password_reset_unique,unique" json:"user_id"`
	User         User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user"`
	HashedSecret string    `gorm:"not null;default:null" json:"hashed_secret" validate:"required"`
	CreatedAt    time.Time `json:"created_at"`
}

type PasswordResetValidateBody struct {
	Code        string `json:"code" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}
