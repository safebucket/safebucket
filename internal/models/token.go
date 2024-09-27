package models

import "github.com/golang-jwt/jwt/v5"

type UserClaims struct {
	Email  string `json:"email"`
	Issuer string `json:"iss"`
	Aud    string `json:"aud"`
	jwt.RegisteredClaims
}
