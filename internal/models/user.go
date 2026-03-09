package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

type User struct {
	ID             uuid.UUID      `gorm:"default:(-)"                                              json:"id"`
	FirstName      string         `gorm:"default:null"                                             json:"first_name"`
	LastName       string         `gorm:"default:null"                                             json:"last_name"`
	Email          string         `gorm:"not null;default:null;uniqueIndex:idx_email_provider_key" json:"email"`
	HashedPassword string         `gorm:"default:null"                                             json:"-"`
	IsInitialized  bool           `gorm:"not null;default:false"                                   json:"is_initialized"`
	ProviderType   ProviderType   `gorm:"not null"                                                 json:"provider_type"`
	ProviderKey    string         `gorm:"not null;uniqueIndex:idx_email_provider_key"              json:"provider_key"`
	Role           Role           `gorm:"not null"                                                 json:"role"`
	CreatedAt      time.Time      `                                                                json:"created_at"`
	UpdatedAt      time.Time      `                                                                json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index"                                                    json:"-"`
	MFADevices     []MFADevice    `gorm:"foreignKey:UserID"                                        json:"-"`
}

func (u *User) HasMFAEnabled() bool {
	for _, d := range u.MFADevices {
		if d.IsVerified {
			return true
		}
	}
	return false
}

func (u *User) GetVerifiedDevices() []MFADevice {
	var verified []MFADevice
	for _, d := range u.MFADevices {
		if d.IsVerified {
			verified = append(verified, d)
		}
	}
	return verified
}

func (u *User) GetDefaultDevice() *MFADevice {
	for i := range u.MFADevices {
		if u.MFADevices[i].IsDefault && u.MFADevices[i].IsVerified {
			return &u.MFADevices[i]
		}
	}
	return nil
}

type UserActivity struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
}

func (u *User) ToActivity() UserActivity {
	return UserActivity{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
	}
}

type UserCreateBody struct {
	FirstName string `json:"first_name" validate:"omitempty,max=100"`
	LastName  string `json:"last_name"  validate:"omitempty,max=100"`
	Email     string `json:"email"      validate:"required,omitempty,email,max=254"`
	Password  string `json:"password"   validate:"required,min=8,max=72"`
}

type UserUpdateBody struct {
	FirstName   string `json:"first_name"   validate:"omitempty,max=100"`
	LastName    string `json:"last_name"    validate:"omitempty,max=100"`
	OldPassword string `json:"old_password" validate:"omitempty,required_with=NewPassword,max=72"`
	NewPassword string `json:"new_password" validate:"omitempty,min=8,max=72"`
}

type UserStatsResponse struct {
	TotalFiles   int `json:"total_files"`
	TotalBuckets int `json:"total_buckets"`
}
