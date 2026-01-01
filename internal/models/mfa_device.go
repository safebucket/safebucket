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

// MaxMFADevicesPerUser is the maximum number of MFA devices allowed per user.
const MaxMFADevicesPerUser = 5

// TOTPCodeTTL is the time-to-live for TOTP code replay protection (in seconds).
const TOTPCodeTTL = 90

// MFAMaxAttempts is the maximum number of failed MFA verification attempts before lockout.
const MFAMaxAttempts = 5

// MFALockoutSeconds is the lockout duration after max failed MFA attempts (in seconds).
const MFALockoutSeconds = 900

// MFADevice represents an MFA device associated with a user.
type MFADevice struct {
	ID              uuid.UUID     `gorm:"type:uuid;primarykey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID     `gorm:"type:uuid;not null;index"                       json:"user_id"`
	Name            string        `gorm:"type:varchar(100);not null"                     json:"name"`
	Type            MFADeviceType `gorm:"type:mfa_device_type;not null;default:'totp'"   json:"type"`
	SecretEncrypted string        `gorm:"not null"                                       json:"-"`
	IsDefault       bool          `gorm:"column:is_default;not null;default:false"       json:"is_default"`
	IsVerified      bool          `gorm:"not null;default:false"                         json:"is_verified"`
	CreatedAt       time.Time     `                                                      json:"created_at"`
	UpdatedAt       time.Time     `                                                      json:"updated_at"`
	VerifiedAt      *time.Time    `                                                      json:"verified_at,omitempty"`
	LastUsedAt      *time.Time    `                                                      json:"last_used_at,omitempty"`
}

// TableName returns the table name for the MFADevice model.
func (*MFADevice) TableName() string {
	return "mfa_devices"
}

// MFADeviceResponse is the public response for an MFA device (no secrets).
type MFADeviceResponse struct {
	ID         uuid.UUID     `json:"id"`
	Name       string        `json:"name"`
	Type       MFADeviceType `json:"type"`
	IsDefault  bool          `json:"is_default"`
	IsVerified bool          `json:"is_verified"`
	CreatedAt  time.Time     `json:"created_at"`
	VerifiedAt *time.Time    `json:"verified_at,omitempty"`
	LastUsedAt *time.Time    `json:"last_used_at,omitempty"`
}

// ToResponse converts MFADevice to public response (without secrets).
func (d *MFADevice) ToResponse() MFADeviceResponse {
	return MFADeviceResponse{
		ID:         d.ID,
		Name:       d.Name,
		Type:       d.Type,
		IsDefault:  d.IsDefault,
		IsVerified: d.IsVerified,
		CreatedAt:  d.CreatedAt,
		VerifiedAt: d.VerifiedAt,
		LastUsedAt: d.LastUsedAt,
	}
}

// MFADevicesListResponse wraps device list with user MFA status.
type MFADevicesListResponse struct {
	Devices     []MFADeviceResponse `json:"devices"`
	MFAEnabled  bool                `json:"mfa_enabled"`
	DeviceCount int                 `json:"device_count"`
	MaxDevices  int                 `json:"max_devices"`
}

// MFADeviceSetupBody is used when setting up a new MFA device.
type MFADeviceSetupBody struct {
	Name string `json:"name" validate:"omitempty,max=100"`
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
