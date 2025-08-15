package models

type Configuration struct {
	Platform PlatformConfiguration `mapstructure:"platform" validate:"required"`
	Database DatabaseConfiguration `mapstructure:"database" validate:"required"`
	JWT      JWTConfiguration      `mapstructure:"jwt" validate:"required"`
	Cors     CorsConfiguration     `mapstructure:"cors" validate:"required"`
	Auth     AuthConfiguration     `mapstructure:"auth" validate:"required"`
	Cache    CacheConfiguration    `mapstructure:"cache" validate:"required"`
	Storage  StorageConfiguration  `mapstructure:"storage" validate:"required"`
	Admin    AdminConfiguration    `mapstructure:"admin" validate:"required"`
	Events   EventsConfiguration   `mapstructure:"events" validate:"required"`
	Mailer   MailerConfiguration   `mapstructure:"mailer" validate:"required"`
	Activity ActivityConfiguration `mapstructure:"activity" validate:"required"`
}

type PlatformConfiguration struct {
	ApiUrl         string   `mapstructure:"api_url" validate:"required"`
	WebUrl         string   `mapstructure:"web_url" validate:"required"`
	TrustedProxies []string `mapstructure:"trusted_proxies" validate:"required"`
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
	Providers map[string]ProviderConfiguration `mapstructure:"providers"`
}

type ProviderConfiguration struct {
	Name                 string               `mapstructure:"name" validate:"required"`
	ClientId             string               `mapstructure:"client_id" validate:"required"`
	ClientSecret         string               `mapstructure:"client_secret" validate:"required"`
	Issuer               string               `mapstructure:"issuer" validate:"required"`
	SharingConfiguration SharingConfiguration `mapstructure:"sharing" validate:"dive"`
}

type SharingConfiguration struct {
	Enabled        bool     `mapstructure:"enabled" default:"true"`
	AllowedDomains []string `mapstructure:"allowed_domains" validate:"dive,hostname_rfc1123"`
}

type CacheConfiguration struct {
	Type   string                    `mapstructure:"type" validate:"required,oneof=redis valkey"`
	Redis  *RedisCacheConfiguration  `mapstructure:"redis" validate:"required_if=Type redis"`
	Valkey *ValkeyCacheConfiguration `mapstructure:"valkey" validate:"required_if=Type valkey"`
}

type RedisCacheConfiguration struct {
	Hosts    []string `mapstructure:"hosts" validate:"required"`
	Port     int32    `mapstructure:"port" validate:"gte=80,lte=65535"`
	Password string   `mapstructure:"password" validate:"required"`
}

type ValkeyCacheConfiguration struct {
	Hosts    []string `mapstructure:"hosts" validate:"required"`
	Port     int32    `mapstructure:"port" validate:"gte=80,lte=65535"`
	Password string   `mapstructure:"password" validate:"required"`
}

type StorageConfiguration struct {
	Type string `mapstructure:"type" validate:"required,oneof=minio gcp aws"`

	Minio *MinioStorageConfiguration `mapstructure:"minio" validate:"required_if=Type minio"`
	GCP   *GCPConfiguration          `mapstructure:"gcp" validate:"required_if=Type gcp"`
	AWS   *AWSConfiguration          `mapstructure:"aws" validate:"required_if=Type aws"`
}

type MinioStorageConfiguration struct {
	BucketName   string `mapstructure:"bucket_name" default:"safebucket"`
	Endpoint     string `mapstructure:"endpoint" validate:"required"`
	ClientId     string `mapstructure:"client_id" validate:"required"`
	ClientSecret string `mapstructure:"client_secret" validate:"required"`

	Type      string                 `mapstructure:"type" validate:"required,oneof=jetstream"`
	Jetstream *JetStreamEventsConfig `mapstructure:"jetstream"`
}

type GCPConfiguration struct {
	BucketName       string `mapstructure:"bucket_name" default:"safebucket"`
	TopicName        string `mapstructure:"topic_name" validate:"required"`
	ProjectID        string `mapstructure:"project_id" validate:"required"`
	SubscriptionName string `mapstructure:"subscription_name" validate:"required"`
}

type AWSConfiguration struct {
	BucketName string `mapstructure:"bucket_name" default:"safebucket"`
	SQSName    string `mapstructure:"sqs_name" validate:"required"`
}

type AdminConfiguration struct {
	Username string `mapstructure:"username" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
}

type EventsConfiguration struct {
	Type string `mapstructure:"type" validate:"required,oneof=jetstream gcp aws"`

	Jetstream *JetStreamEventsConfig `mapstructure:"jetstream" validate:"required_if=Type jetstream"`
	GCP       *GCPConfiguration      `mapstructure:"gcp" validate:"required_if=Type gcp"`
	AWS       *AWSConfiguration      `mapstructure:"aws" validate:"required_if=Type aws"`
}

type JetStreamEventsConfig struct {
	TopicName string `mapstructure:"topic_name" validate:"required"`
	Host      string `mapstructure:"host" validate:"required"`
	Port      string `mapstructure:"port" validate:"required"`
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
