package models

import "github.com/google/uuid"

// MFASetupResponse is returned when initiating MFA setup.
//
// Deprecated: Use MFADeviceSetupResponse for multi-device support.
type MFASetupResponse struct {
	Secret    string `json:"secret"`      // Base32 encoded secret for manual entry
	QRCodeURI string `json:"qr_code_uri"` // otpauth:// URI for QR code generation
	Issuer    string `json:"issuer"`      // Application name (SafeBucket)
}

// MFAVerifyBody is used to verify a TOTP code during MFA setup.
//
// Deprecated: Use MFADeviceVerifyBody for multi-device support.
type MFAVerifyBody struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

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
