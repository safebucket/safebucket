package models

type ProviderType string

const (
	LocalProviderType ProviderType = "local"
	OIDCProviderType  ProviderType = "oidc"
)

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
	Id      string       `json:"id"`
	Name    string       `json:"name"`
	Type    ProviderType `json:"type"`
	Domains []string     `json:"domains"`
}
