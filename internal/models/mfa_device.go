package models

import (
	"time"

	"github.com/google/uuid"
)

// MFADeviceType represents the type of MFA device.
type MFADeviceType string

const (
	MFADeviceTypeTOTP MFADeviceType = "totp"
)

// MFADevice represents an MFA device associated with a user.
type MFADevice struct {
	ID              uuid.UUID     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID     `gorm:"type:uuid;not null;index"                       json:"user_id"`
	Name            string        `gorm:"type:varchar(100);not null"                     json:"name"`
	Type            MFADeviceType `gorm:"type:mfa_device_type;not null;default:'totp'"   json:"type"`
	EncryptedSecret string        `gorm:"not null"                                       json:"-"`
	IsDefault       bool          `gorm:"column:is_default;not null;default:false"       json:"is_default"`
	IsVerified      bool          `gorm:"not null;default:false"                         json:"is_verified"`
	CreatedAt       time.Time     `                                                      json:"created_at"`
	UpdatedAt       time.Time     `                                                      json:"updated_at"`
	VerifiedAt      *time.Time    `                                                      json:"verified_at,omitempty"`
	LastUsedAt      *time.Time    `                                                      json:"last_used_at,omitempty"`
}

// MFADevicesListResponse wraps device list with user MFA status.
type MFADevicesListResponse struct {
	Devices     []MFADevice `json:"devices"`
	MFAEnabled  bool        `json:"mfa_enabled"`
	DeviceCount int         `json:"device_count"`
	MaxDevices  int         `json:"max_devices"`
}

// MFADeviceSetupBody is used to initiate MFA setup.
type MFADeviceSetupBody struct {
	Name     string `json:"name"     validate:"required,min=1,max=50"`
	Password string `json:"password" validate:"omitempty"`
}

// MFADeviceSetupResponse is returned when initiating device setup.
type MFADeviceSetupResponse struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Secret    string    `json:"secret"`
	QRCodeURI string    `json:"qr_code_uri"`
	Issuer    string    `json:"issuer"`
}

// MFADeviceVerifyBody is used to verify and enable a new device.
type MFADeviceVerifyBody struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

// MFADeviceUpdateBody is used to update device properties.
type MFADeviceUpdateBody struct {
	Name      *string `json:"name"       validate:"omitempty,min=1,max=100"`
	IsDefault *bool   `json:"is_default" validate:"omitempty"`
}

// MFADeviceRemoveBody is used when removing an MFA device (requires password for security).
type MFADeviceRemoveBody struct {
	Password string `json:"password" validate:"required,min=1"`
}
