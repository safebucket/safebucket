package models

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserClaims struct {
	jwt.RegisteredClaims

	Email    string    `json:"email"`
	UserID   uuid.UUID `json:"user_id"`
	Role     Role      `json:"role"`
	Issuer   string    `json:"iss"`
	Aud      string    `json:"aud"`
	Provider string    `json:"provider"`
}

func (u *UserClaims) Valid() bool {
	return u.UserID.String() != ""
}

type UserClaimKey struct{}
