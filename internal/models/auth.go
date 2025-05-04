package models

import "api/internal/rbac"

type AuthLogin struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthLoginResponse struct {
	AccessToken  string `json:"access_token" validate:"required"`
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthVerify struct {
	AccessToken string `json:"access_token" validate:"required"`
}

type AuthVerifyResponse struct {
	Valid bool `json:"valid"`
}

type AuthRefresh struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type AuthRefreshResponse struct {
	AccessToken string `json:"access_token" validate:"required"`
}

type ProviderResponse struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type AuthzParameters struct {
	ObjectIdIndex int
	ObjectType    rbac.Resource
	Action        rbac.Action
}
