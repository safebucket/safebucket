package models

import (
	"time"

	"github.com/google/uuid"
)

type ChallengeType string

const (
	ChallengeTypeInvite        ChallengeType = "invite"
	ChallengeTypePasswordReset ChallengeType = "password_reset"
)

type Challenge struct {
	ID           uuid.UUID     `gorm:"default:(-)"                                                      json:"id"`
	Type         ChallengeType `gorm:"not null;index:idx_challenge_type"                                json:"type"                 validate:"required,oneof=invite password_reset"`
	HashedSecret string        `gorm:"not null;default:null"                                            json:"hashed_secret"        validate:"required"`
	AttemptsLeft int           `gorm:"not null;default:3"                                               json:"attempts_left"`
	ExpiresAt    *time.Time    `gorm:"index"                                                            json:"expires_at,omitempty"`
	CreatedAt    time.Time     `                                                                        json:"created_at"`
	DeletedAt    *time.Time    `gorm:"index"                                                            json:"deleted_at,omitempty"`
	InviteID     *uuid.UUID    `gorm:"index:idx_challenge_invite,unique"                                json:"invite_id,omitempty"`
	Invite       *Invite       `gorm:"foreignKey:InviteID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"invite,omitempty"`
	UserID       *uuid.UUID    `gorm:"index:idx_challenge_user,unique"                                  json:"user_id,omitempty"`
	User         *User         `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"   json:"user,omitempty"`
}
