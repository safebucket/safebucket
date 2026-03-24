package models

import (
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
