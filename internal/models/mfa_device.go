package models

import (
	"time"

	"github.com/google/uuid"
)

type MFADeviceType string

const (
	MFADeviceTypeTOTP MFADeviceType = "totp"
)

type MFADevice struct {
	ID              uuid.UUID     `gorm:"default:(-)"                              json:"id"`
	UserID          uuid.UUID     `gorm:"not null;index"                           json:"user_id"`
	Name            string        `gorm:"type:varchar(100);not null"               json:"name"`
	Type            MFADeviceType `gorm:"not null;default:'totp'"                  json:"type"`
	EncryptedSecret string        `gorm:"not null"                                 json:"-"`
	IsDefault       bool          `gorm:"column:is_default;not null;default:false" json:"is_default"`
	IsVerified      bool          `gorm:"not null;default:false"                   json:"-"`
	CreatedAt       time.Time     `                                                json:"created_at"`
	UpdatedAt       time.Time     `                                                json:"updated_at"`
	VerifiedAt      *time.Time    `                                                json:"verified_at,omitempty"`
	LastUsedAt      *time.Time    `                                                json:"last_used_at,omitempty"`
}

type MFADevicesListResponse struct {
	Devices []MFADevice `json:"devices"`
}

type MFADeviceSetupBody struct {
	Name     string `json:"name"     validate:"required,min=1,max=50"`
	Password string `json:"password" validate:"omitempty"`
}

type MFADeviceSetupResponse struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Secret    string    `json:"secret"`
	QRCodeURI string    `json:"qr_code_uri"`
	Issuer    string    `json:"issuer"`
}

type MFADeviceVerifyBody struct {
	Code string `json:"code" validate:"required,len=6,numeric"`
}

type MFADeviceUpdateBody struct {
	Name      *string `json:"name"       validate:"omitempty,min=1,max=100"`
	IsDefault *bool   `json:"is_default" validate:"omitempty"`
}

type MFADeviceRemoveBody struct {
	Password string `json:"password" validate:"required,min=1"`
}

type MFADeviceActivity struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (d *MFADevice) ToActivity() MFADeviceActivity {
	return MFADeviceActivity{
		ID:   d.ID,
		Name: d.Name,
	}
}
