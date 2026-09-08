package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserClaims struct {
	jwt.RegisteredClaims

	Email       string     `json:"email"`
	UserID      uuid.UUID  `json:"user_id"`
	Role        Role       `json:"role"`
	Provider    string     `json:"provider"`
	MFA         bool       `json:"mfa"`
	SID         string     `json:"sid,omitempty"`
	ChallengeID *uuid.UUID `json:"challenge_id,omitempty"`
	TokenID     *uuid.UUID `json:"token_id,omitempty"`
}

func (u *UserClaims) Valid() bool {
	return u.UserID.String() != ""
}

// AudienceString returns the first audience claim or an empty string.
func (u *UserClaims) AudienceString() string {
	if len(u.Audience) > 0 {
		return u.Audience[0]
	}
	return ""
}

type UserClaimKey struct{}

type QueryKey struct{}

type ShareClaims struct {
	jwt.RegisteredClaims

	ShareID uuid.UUID `json:"share_id"`
}

type APIToken struct {
	ID         uuid.UUID  `gorm:"default:(-)"    json:"id"`
	TokenHash  string     `gorm:"not null;index" json:"-"`
	UserID     uuid.UUID  `gorm:"not null;index" json:"user_id"`
	Name       string     `gorm:"not null"       json:"name"`
	ExpiresAt  time.Time  `gorm:"not null"       json:"expires_at"`
	LastUsedAt *time.Time `                      json:"last_used_at,omitempty"`
	CreatedBy  *uuid.UUID `                      json:"created_by,omitempty"`
	CreatedAt  time.Time  `                      json:"created_at"`
	RevokedAt  *time.Time `                      json:"revoked_at,omitempty"`
	DeletedAt  *time.Time `gorm:"index"          json:"-"`
}

type TokenCreateBody struct {
	Name       string `json:"name"        validate:"required,min=1,max=100"`
	ExpiryDays int    `json:"expiry_days" validate:"required,gte=1,lte=365"`
}

type TokenCreateResponse struct {
	APIToken

	Secret string `json:"secret"`
}

type TokenActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (t *APIToken) ToActivity() TokenActivity {
	return TokenActivity{
		ID:   t.ID,
		Name: t.Name,
	}
}
