package models

import "github.com/google/uuid"

type MFALoginVerifyBody struct {
	DeviceID *uuid.UUID `json:"device_id" validate:"omitempty"`
	Code     string     `json:"code"      validate:"required,len=6,numeric"`
}

type MFAResetRequestBody struct {
	Password string `json:"password" validate:"required"`
}

type MFAResetVerifyBody struct {
	Code string `json:"code" validate:"required,len=6,alphanum"`
}

type MFAResetRequestResponse struct {
	ChallengeID string `json:"challenge_id"`
}
