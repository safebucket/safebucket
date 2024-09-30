package models

type Configuration struct {
	Database      DatabaseConfiguration      `mapstructure:"database" validate:"required,dive"`
	JWT           JWTConfiguration           `json:"jwt" validate:"required,dive"`
	Cors          CorsConfiguration          `json:"cors" validate:"required,dive"`
	AuthProviders AuthProvidersConfiguration `mapstructure:"auth_providers" validate:"required,dive"`
}

type DatabaseConfiguration struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int32  `mapstructure:"port" validate:"gte=80,lte=65535" `
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name     string `mapstructure:"name" validate:"required"`
	SSLMode  string `mapstructure:"sslmode"`
}

type JWTConfiguration struct {
	Secret string `mapstructure:"secret" validate:"required"`
}

type CorsConfiguration struct {
	AllowedOrigins []string `mapstructure:"allowed_origins" validate:"required"`
}

type AuthProvidersConfiguration struct {
	GoogleClientId     string `mapstructure:"google_client_id" validate:"required"`
	GoogleClientSecret string `mapstructure:"google_client_secret" validate:"required"`
}
