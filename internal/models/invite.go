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

type InviteChallengeValidateBody struct {
	Code string `json:"code" validate:"required"` // TODO: validate value
}

type PasswordResetRequestBody struct {
	Email string `json:"email" validate:"required,email"`
}

type PasswordResetValidateBody struct {
	Code        string `json:"code" validate:"required,len=6,alphanum"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ChallengeType defines the type of challenge
type ChallengeType string

const (
	ChallengeTypeInvite        ChallengeType = "invite"
	ChallengeTypePasswordReset ChallengeType = "password_reset"
)

// Challenge is a unified table for all challenge types (invites and password resets)
type Challenge struct {
	ID           uuid.UUID     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	Type         ChallengeType `gorm:"type:challenge_type;not null;index:idx_challenge_type" json:"type" validate:"required,oneof=invite password_reset"`
	HashedSecret string        `gorm:"not null;default:null" json:"hashed_secret" validate:"required"`
	AttemptsLeft int           `gorm:"not null;default:3" json:"attempts_left"`
	ExpiresAt    *time.Time    `gorm:"index" json:"expires_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	DeletedAt    *time.Time    `gorm:"index" json:"deleted_at,omitempty"`
	InviteID     *uuid.UUID    `gorm:"type:uuid;index:idx_challenge_invite,unique" json:"invite_id,omitempty"`
	Invite       *Invite       `gorm:"foreignKey:InviteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"invite,omitempty"`
	UserID       *uuid.UUID    `gorm:"type:uuid;index:idx_challenge_user,unique" json:"user_id,omitempty"`
	User         *User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"user,omitempty"`
}
