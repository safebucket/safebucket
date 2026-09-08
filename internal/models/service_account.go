package models

import (
	"time"

	"github.com/google/uuid"
)

type ServiceAccountCreateBody struct {
	Name    string `json:"name"     validate:"required,min=1,max=100"`
	Role    Role   `json:"role"     validate:"required,oneof=guest user admin"`
	IsAdmin bool   `json:"is_admin"`
}

type ServiceAccountUpdateBody struct {
	Name    *string `json:"name"     validate:"omitempty,min=1,max=100"`
	Role    *Role   `json:"role"     validate:"omitempty,oneof=guest user admin"`
	IsAdmin bool    `json:"is_admin"`
}

type ServiceAccountResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
}

func NewServiceAccountResponse(user User) ServiceAccountResponse {
	return ServiceAccountResponse{
		ID:        user.ID,
		Name:      user.FirstName,
		Email:     user.Email,
		Role:      user.Role,
		IsAdmin:   user.Role == RoleAdmin,
		CreatedAt: user.CreatedAt,
	}
}
