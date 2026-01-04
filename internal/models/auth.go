package models

type ProviderType string

const (
	LocalProviderType ProviderType = "local"
	OIDCProviderType  ProviderType = "oidc"
)

type AuthLoginBody struct {
	Email    string `json:"email"    validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

type AuthLoginResponse struct {
	AccessToken      string      `json:"access_token,omitempty"`
	RefreshToken     string      `json:"refresh_token,omitempty"`
	MFARequired      bool        `json:"mfa_required"`
	MFAToken         string      `json:"mfa_token,omitempty"`
	MFASetupRequired bool        `json:"mfa_setup_required,omitempty"`
	Devices          []MFADevice `json:"devices,omitempty"`
}

type AuthVerifyBody struct {
	AccessToken string `json:"access_token" validate:"required,max=2048"`
}

type AuthRefreshBody struct {
	RefreshToken string `json:"refresh_token" validate:"required,max=2048"`
}

type AuthRefreshResponse struct {
	AccessToken string `json:"access_token" validate:"required"`
}

type ProviderResponse struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Type    ProviderType `json:"type"`
	Domains []string     `json:"domains"`
}

type PasswordResetRequestBody struct {
	Email string `json:"email" validate:"required,email,max=254"`
}

type PasswordResetValidateBody struct {
	Code        string `json:"code"         validate:"required,len=6,alphanum"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}
