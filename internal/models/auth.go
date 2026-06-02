package models

import "github.com/google/uuid"

type ProviderType string

const (
	LocalProviderType ProviderType = "local"
	OIDCProviderType  ProviderType = "oidc"
	LDAPProviderType  ProviderType = "ldap"
)

type AuthLoginBody struct {
	Email    string `json:"email"    validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

type AuthLoginResponse struct {
	MFARequired bool       `json:"mfa_required"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

type AuthVerifyBody struct {
	AccessToken string `json:"access_token" validate:"required,max=2048"`
}

type AuthMeResponse struct {
	UserID          uuid.UUID `json:"user_id"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	AuthProvider    string    `json:"auth_provider"`
	MFA             bool      `json:"mfa"`
	MFADevicesCount int       `json:"mfa_devices_count"`
}

type ProviderResponse struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Type    ProviderType `json:"type"`
	Domains []string     `json:"domains"`
}

type PasswordResetRequestBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

type PasswordResetValidateBody struct {
	Code string `json:"code" validate:"required,len=6,alphanum"`
}

type PasswordResetCompleteBody struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}
