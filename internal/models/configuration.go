package models

type Configuration struct {
	Platform PlatformConfiguration `mapstructure:"platform" validate:"required,dive"`
	Database DatabaseConfiguration `mapstructure:"database" validate:"required,dive"`
	JWT      JWTConfiguration      `json:"jwt" validate:"required,dive"`
	Cors     CorsConfiguration     `json:"cors" validate:"required,dive"`
	Auth     AuthConfiguration     `mapstructure:"auth" validate:"required,dive"`
	Redis    RedisConfiguration    `json:"redis" validate:"required,dive"`
	Storage  StorageConfiguration  `mapstructure:"storage" validate:"required,dive"`
	Admin    AdminConfiguration    `mapstructure:"admin" validate:"required,dive"`
	Events   EventsConfiguration   `mapstructure:"events" validate:"required,dive"`
	Mailer   MailerConfiguration   `mapstructure:"mailer" validate:"required,dive"`
	Activity ActivityConfiguration `mapstructure:"activity" validate:"required,dive"`
}

type PlatformConfiguration struct {
	ApiUrl string `mapstructure:"api_url" validate:"required"`
	WebUrl string `mapstructure:"web_url" validate:"required"`
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

type AuthConfiguration struct {
	Providers map[string]ProviderConfiguration `mapstructure:"providers" validate:"dive"`
}

type ProviderConfiguration struct {
	Name         string `mapstructure:"name" validate:"required"`
	ClientId     string `mapstructure:"client_id" validate:"required"`
	ClientSecret string `mapstructure:"client_secret" validate:"required"`
	Issuer       string `mapstructure:"issuer" validate:"required"`
}

type RedisConfiguration struct {
	Hosts    []string `mapstructure:"hosts" validate:"required"`
	Port     int32    `mapstructure:"port" validate:"gte=80,lte=65535"`
	Password string   `mapstructure:"password" validate:"required"`
}

type StorageConfiguration struct {
	BucketName   string `mapstructure:"bucket_name" default:"safebucket"`
	Type         string `mapstructure:"type" validate:"required,oneof=s3 gcp"`
	Endpoint     string `mapstructure:"endpoint" validate:"required"`
	ClientId     string `mapstructure:"client_id" validate:"required"`
	ClientSecret string `mapstructure:"client_secret" validate:"required"`
}

type AdminConfiguration struct {
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
}

type EventsConfiguration struct {
	Type string `mapstructure:"type" validate:"required,oneof=jetstream"`
	Host string `mapstructure:"host" validate:"required"`
	Port string `mapstructure:"port" validate:"required"`
}

type MailerConfiguration struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Sender   string `mapstructure:"sender" validate:"required"`
}

type ActivityConfiguration struct {
	Level    string `mapstructure:"level"`
	Type     string `mapstructure:"type" validate:"required,oneof=loki"`
	Endpoint string `mapstructure:"endpoint" validate:"required"`
}
