package models

type Configuration struct {
	App      AppConfiguration      `mapstructure:"app" validate:"required"`
	Database DatabaseConfiguration `mapstructure:"database" validate:"required"`
	Auth     AuthConfiguration     `mapstructure:"auth" validate:"required"`
	Cache    CacheConfiguration    `mapstructure:"cache" validate:"required"`
	Storage  StorageConfiguration  `mapstructure:"storage" validate:"required"`
	Events   EventsConfiguration   `mapstructure:"events" validate:"required"`
	Notifier NotifierConfiguration `mapstructure:"notifier" validate:"required"`
	Activity ActivityConfiguration `mapstructure:"activity" validate:"required"`
}

type AppConfiguration struct {
	AdminEmail     string              `mapstructure:"admin_email" validate:"required,email"`
	AdminPassword  string              `mapstructure:"admin_password" validate:"required"`
	ApiUrl         string              `mapstructure:"api_url" validate:"required"`
	AllowedOrigins []string            `mapstructure:"allowed_origins" validate:"required"`
	JWTSecret      string              `mapstructure:"jwt_secret" validate:"required"`
	LogLevel       string              `mapstructure:"log_level" validate:"oneof=debug info warn error fatal panic" default:"info"`
	Port           int                 `mapstructure:"port" validate:"gte=80,lte=65535" default:"8080"`
	StaticFiles    StaticConfiguration `mapstructure:"static_files"`
	TrustedProxies []string            `mapstructure:"trusted_proxies" validate:"required"`
	WebUrl         string              `mapstructure:"web_url" validate:"required"`
}

type DatabaseConfiguration struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int32  `mapstructure:"port" validate:"gte=80,lte=65535" default:"5432"`
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name     string `mapstructure:"name" validate:"required"`
	SSLMode  string `mapstructure:"sslmode"`
}

type AuthConfiguration struct {
	Providers map[string]ProviderConfiguration `mapstructure:"providers" validate:"omitempty,dive"`
}

type ProviderConfiguration struct {
	Name                 string               `mapstructure:"name" validate:"required_if=Type oidc"`
	Type                 string               `mapstructure:"type" validate:"required,oneof=local oidc"`
	OIDC                 OIDCConfiguration    `mapstructure:"oidc" validate:"required_if=Type oidc"`
	SharingConfiguration SharingConfiguration `mapstructure:"sharing"`
}

type OIDCConfiguration struct {
	ClientId     string `mapstructure:"client_id" validate:"required_if=Type oidc"`
	ClientSecret string `mapstructure:"client_secret" validate:"required_if=Type oidc"`
	Issuer       string `mapstructure:"issuer" validate:"required_if=Type oidc"`
}

type SharingConfiguration struct {
	Allowed        bool     `mapstructure:"allowed" default:"true"`
	AllowedDomains []string `mapstructure:"allowed_domains" validate:"dive,hostname_rfc1123"`
}

type CacheConfiguration struct {
	Type   string                    `mapstructure:"type" validate:"required,oneof=redis valkey"`
	Redis  *RedisCacheConfiguration  `mapstructure:"redis" validate:"required_if=Type redis"`
	Valkey *ValkeyCacheConfiguration `mapstructure:"valkey" validate:"required_if=Type valkey"`
}

type RedisCacheConfiguration struct {
	Hosts         []string `mapstructure:"hosts"`
	Password      string   `mapstructure:"password"`
	TLSEnabled    bool     `mapstructure:"tls_enabled"`
	TLSServerName string   `mapstructure:"tls_server_name"`
}

type ValkeyCacheConfiguration struct {
	Hosts         []string `mapstructure:"hosts"`
	Password      string   `mapstructure:"password"`
	TLSEnabled    bool     `mapstructure:"tls_enabled"`
	TLSServerName string   `mapstructure:"tls_server_name"`
}

type StorageConfiguration struct {
	Type         string                     `mapstructure:"type" validate:"required,oneof=minio gcp aws"`
	Minio        *MinioStorageConfiguration `mapstructure:"minio" validate:"required_if=Type minio"`
	CloudStorage *CloudStorage              `mapstructure:"gcp" validate:"required_if=Type gcp"`
	S3           *S3Configuration           `mapstructure:"aws" validate:"required_if=Type aws"`
}

type MinioStorageConfiguration struct {
	BucketName   string `mapstructure:"bucket_name" validate:"required"`
	Endpoint     string `mapstructure:"endpoint" validate:"required"`
	ClientId     string `mapstructure:"client_id" validate:"required"`
	ClientSecret string `mapstructure:"client_secret" validate:"required"`
}

type CloudStorage struct {
	BucketName string `mapstructure:"bucket_name" validate:"required"`
	ProjectID  string `mapstructure:"project_id" validate:"required"`
}

type S3Configuration struct {
	BucketName string `mapstructure:"bucket_name" validate:"required"`
}

type QueueConfig struct {
	Name string `mapstructure:"name" validate:"required"`
}

type EventsConfiguration struct {
	Type      string                 `mapstructure:"type" validate:"required,oneof=jetstream gcp aws"`
	Queues    map[string]QueueConfig `mapstructure:"queues" validate:"required"`
	Jetstream *JetStreamEventsConfig `mapstructure:"jetstream" validate:"required_if=Type jetstream"`
	PubSub    *PubSubConfiguration   `mapstructure:"gcp" validate:"required_if=Type gcp"`
}

type PubSubConfiguration struct {
	ProjectID          string `mapstructure:"project_id" validate:"required"`
	SubscriptionSuffix string `mapstructure:"subscription_suffix" default:"-sub"`
}

type JetStreamEventsConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port string `mapstructure:"port" validate:"required"`
}

type MailerConfiguration struct {
	Host          string `mapstructure:"host" validate:"required"`
	Port          int    `mapstructure:"port" validate:"required"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	Sender        string `mapstructure:"sender" validate:"required"`
	EnableTLS     bool   `mapstructure:"enable_tls" default:"true"`
	SkipVerifyTLS bool   `mapstructure:"skip_verify_tls" default:"false"`
}

type NotifierConfiguration struct {
	Type string               `mapstructure:"type" validate:"required,oneof=smtp"`
	SMTP *MailerConfiguration `mapstructure:"smtp" validate:"required_if=Type smtp"`
}

type ActivityConfiguration struct {
	Level string            `mapstructure:"level"`
	Type  string            `mapstructure:"type" validate:"required,oneof=loki"`
	Loki  LokiConfiguration `mapstructure:"loki" validate:"required_if=Type loki"`
}

type LokiConfiguration struct {
	Endpoint string `mapstructure:"endpoint" validate:"required,http_url"`
}

type StaticConfiguration struct {
	Enabled   bool   `mapstructure:"enabled" default:"true"`
	Directory string `mapstructure:"directory" default:"web/dist"`
}
