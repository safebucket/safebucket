package models

import "github.com/google/uuid"

// MFALoginVerifyBody is used to verify MFA during login.
type MFALoginVerifyBody struct {
	MFAToken string     `json:"mfa_token" validate:"required"`
	DeviceID *uuid.UUID `json:"device_id" validate:"omitempty"`
	Code     string     `json:"code"      validate:"required,len=6,numeric"`
}

// MFAResetRequestBody is used to initiate an MFA reset.
type MFAResetRequestBody struct {
	Password string `json:"password" validate:"required"`
}

// MFAResetVerifyBody is used to verify and complete MFA reset.
type MFAResetVerifyBody struct {
	Code string `json:"code" validate:"required,len=6,alphanum"`
}

// MFAResetRequestResponse is returned when MFA reset is initiated.
type MFAResetRequestResponse struct {
	ChallengeID string `json:"challenge_id"`
}
