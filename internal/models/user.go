package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents the platform-wide access level of a user.
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

type User struct {
	ID             uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()"           json:"id"`
	FirstName      string         `gorm:"default:null"                                             json:"first_name"`
	LastName       string         `gorm:"default:null"                                             json:"last_name"`
	Email          string         `gorm:"not null;default:null;uniqueIndex:idx_email_provider_key" json:"email"`
	HashedPassword string         `gorm:"default:null"                                             json:"-"`
	IsInitialized  bool           `gorm:"not null;default:false"                                   json:"is_initialized"`
	ProviderType   ProviderType   `gorm:"not null;type:provider_type;"                             json:"provider_type"`
	ProviderKey    string         `gorm:"not null;uniqueIndex:idx_email_provider_key"              json:"provider_key"`
	Role           Role           `gorm:"type:role_type;not null;"                                 json:"role"`
	CreatedAt      time.Time      `                                                                json:"created_at"`
	UpdatedAt      time.Time      `                                                                json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index"                                                    json:"-"`
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
