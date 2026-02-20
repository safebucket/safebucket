package models

type Configuration struct {
	App      AppConfiguration      `mapstructure:"app"      validate:"required"`
	Database DatabaseConfiguration `mapstructure:"database" validate:"required"`
	Auth     AuthConfiguration     `mapstructure:"auth"     validate:"required"`
	Cache    CacheConfiguration    `mapstructure:"cache"    validate:"required"`
	Storage  StorageConfiguration  `mapstructure:"storage"  validate:"required"`
	Events   EventsConfiguration   `mapstructure:"events"   validate:"required"`
	Notifier NotifierConfiguration `mapstructure:"notifier" validate:"required"`
	Activity ActivityConfiguration `mapstructure:"activity" validate:"required"`
}

type AppConfiguration struct {
	Profile            string              `mapstructure:"profile"              validate:"oneof=default api worker"`
	AdminEmail         string              `mapstructure:"admin_email"          validate:"required,email"`
	AdminPassword      string              `mapstructure:"admin_password"       validate:"required"`
	APIURL             string              `mapstructure:"api_url"              validate:"required"`
	AllowedOrigins     []string            `mapstructure:"allowed_origins"      validate:"required"`
	JWTSecret          string              `mapstructure:"jwt_secret"           validate:"required"`
	MFAEncryptionKey   string              `mapstructure:"mfa_encryption_key"   validate:"len=32"`
	MFARequired        bool                `mapstructure:"mfa_required"`
	AccessTokenExpiry  int                 `mapstructure:"access_token_expiry"  validate:"gte=1,lte=1440"`
	RefreshTokenExpiry int                 `mapstructure:"refresh_token_expiry" validate:"gte=1,lte=720"`
	MFATokenExpiry     int                 `mapstructure:"mfa_token_expiry"     validate:"gte=1,lte=30"`
	LogLevel           string              `mapstructure:"log_level"            validate:"oneof=debug info warn error fatal panic"`
	Port               int                 `mapstructure:"port"                 validate:"gte=80,lte=65535"`
	StaticFiles        StaticConfiguration `mapstructure:"static_files"`
	TrustedProxies     []string            `mapstructure:"trusted_proxies"      validate:"required"`
	WebURL             string              `mapstructure:"web_url"              validate:"required"`
	TrashRetentionDays int                 `mapstructure:"trash_retention_days" validate:"gte=1,lte=365"`
	MaxUploadSize      int64               `mapstructure:"max_upload_size"      validate:"gte=1"`
}

type DatabaseConfiguration struct {
	Host     string `mapstructure:"host"     validate:"required"`
	Port     int32  `mapstructure:"port"     validate:"gte=80,lte=65535"`
	User     string `mapstructure:"user"     validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	Name     string `mapstructure:"name"     validate:"required"`
	SSLMode  string `mapstructure:"sslmode"`
}

type AuthConfiguration struct {
	Providers map[string]ProviderConfiguration `mapstructure:"providers" validate:"omitempty,dive"`
}

type ProviderConfiguration struct {
	Name                 string               `mapstructure:"name"    validate:"required_if=Type oidc"`
	Type                 ProviderType         `mapstructure:"type"    validate:"required,oneof=local oidc"`
	OIDC                 OIDCConfiguration    `mapstructure:"oidc"    validate:"required_if=Type oidc"`
	Domains              []string             `mapstructure:"domains"`
	SharingConfiguration SharingConfiguration `mapstructure:"sharing"`
}

type OIDCConfiguration struct {
	ClientID     string `mapstructure:"client_id"     validate:"required_if=Type oidc"`
	ClientSecret string `mapstructure:"client_secret" validate:"required_if=Type oidc"`
	Issuer       string `mapstructure:"issuer"        validate:"required_if=Type oidc"`
}

type SharingConfiguration struct {
	Allowed bool     `mapstructure:"allowed"`
	Domains []string `mapstructure:"domains" validate:"dive"`
}

type CacheConfiguration struct {
	Type   string                    `mapstructure:"type"   validate:"required,oneof=redis valkey"`
	Redis  *RedisCacheConfiguration  `mapstructure:"redis"  validate:"required_if=Type redis"`
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
	Type         string                      `mapstructure:"type"   validate:"required,oneof=minio gcp aws rustfs s3"`
	Minio        *MinioStorageConfiguration  `mapstructure:"minio"  validate:"required_if=Type minio"`
	CloudStorage *CloudStorage               `mapstructure:"gcp"    validate:"required_if=Type gcp"`
	AWS          *AWSConfiguration           `mapstructure:"aws"    validate:"required_if=Type aws"`
	RustFS       *RustFSStorageConfiguration `mapstructure:"rustfs" validate:"required_if=Type rustfs"`
	S3           *S3Configuration            `mapstructure:"s3"     validate:"required_if=Type s3"`
}

type MinioStorageConfiguration struct {
	BucketName       string `mapstructure:"bucket_name"       validate:"required"`
	Endpoint         string `mapstructure:"endpoint"          validate:"required"`
	ExternalEndpoint string `mapstructure:"external_endpoint" validate:"required,http_url"`
	ClientID         string `mapstructure:"client_id"         validate:"required"`
	ClientSecret     string `mapstructure:"client_secret"     validate:"required"`
}

type CloudStorage struct {
	BucketName string `mapstructure:"bucket_name" validate:"required"`
	ProjectID  string `mapstructure:"project_id"  validate:"required"`
}

// AWSConfiguration for AWS S3 storage.
// Uses AWS SDK default credential chain (environment variables, shared credentials, IAM roles).
type AWSConfiguration struct {
	BucketName       string `mapstructure:"bucket_name"       validate:"required"`
	ExternalEndpoint string `mapstructure:"external_endpoint"`
}

// S3Configuration for generic S3-compatible providers (Storj, Hetzner, Backblaze B2, Garage).
// This provider assumes NO lifecycle policy or bucket notification support.
type S3Configuration struct {
	BucketName       string `mapstructure:"bucket_name"       validate:"required"`
	Endpoint         string `mapstructure:"endpoint"          validate:"required"`
	ExternalEndpoint string `mapstructure:"external_endpoint" validate:"required,http_url"`
	AccessKey        string `mapstructure:"access_key"        validate:"required"`
	SecretKey        string `mapstructure:"secret_key"        validate:"required"`
	Region           string `mapstructure:"region"`
	// ForcePathStyle uses path-style URLs (endpoint/bucket/key) instead of virtual-hosted style (bucket.endpoint/key).
	// Most S3-compatible providers require this to be true.
	ForcePathStyle bool `mapstructure:"force_path_style"`
	UseTLS         bool `mapstructure:"use_tls"`
}

type RustFSStorageConfiguration struct {
	BucketName       string `mapstructure:"bucket_name"       validate:"required"`
	Endpoint         string `mapstructure:"endpoint"          validate:"required"`
	ExternalEndpoint string `mapstructure:"external_endpoint" validate:"required,http_url"`
	AccessKey        string `mapstructure:"access_key"        validate:"required"`
	SecretKey        string `mapstructure:"secret_key"        validate:"required"`
}

// GetExternalURL returns the external URL for the configured storage provider.
// This URL is used for browser-accessible endpoints (e.g., for CSP headers).
// Returns empty string if no external URL is configured or applicable.
func (s *StorageConfiguration) GetExternalURL() string {
	switch s.Type {
	case "minio":
		if s.Minio != nil {
			return s.Minio.ExternalEndpoint
		}
	case "rustfs":
		if s.RustFS != nil {
			return s.RustFS.ExternalEndpoint
		}
	case "gcp":
		return ""
	case "aws":
		if s.AWS != nil && s.AWS.ExternalEndpoint != "" {
			return s.AWS.ExternalEndpoint
		}
		return ""
	case "s3":
		if s.S3 != nil {
			return s.S3.ExternalEndpoint
		}
	}
	return ""
}

type QueueConfig struct {
	Name string `mapstructure:"name" validate:"required"`
}

type EventsConfiguration struct {
	Type      string                 `mapstructure:"type"      validate:"required,oneof=jetstream gcp aws memory"`
	Queues    map[string]QueueConfig `mapstructure:"queues"    validate:"required"`
	Jetstream *JetStreamEventsConfig `mapstructure:"jetstream" validate:"required_if=Type jetstream"`
	PubSub    *PubSubConfiguration   `mapstructure:"gcp"       validate:"required_if=Type gcp"`
}

type PubSubConfiguration struct {
	ProjectID          string `mapstructure:"project_id"          validate:"required"`
	SubscriptionSuffix string `mapstructure:"subscription_suffix"`
}

type JetStreamEventsConfig struct {
	Host string `mapstructure:"host" validate:"required"`
	Port string `mapstructure:"port" validate:"required"`
}

type MailerConfiguration struct {
	Host          string `mapstructure:"host"            validate:"required"`
	Port          int    `mapstructure:"port"            validate:"required"`
	Username      string `mapstructure:"username"`
	Password      string `mapstructure:"password"`
	Sender        string `mapstructure:"sender"          validate:"required"`
	EnableTLS     bool   `mapstructure:"enable_tls"`
	SkipVerifyTLS bool   `mapstructure:"skip_verify_tls"`
}

type NotifierConfiguration struct {
	Type       string                           `mapstructure:"type"       validate:"required,oneof=smtp filesystem"`
	SMTP       *MailerConfiguration             `mapstructure:"smtp"       validate:"required_if=Type smtp"`
	Filesystem *FilesystemNotifierConfiguration `mapstructure:"filesystem" validate:"required_if=Type filesystem"`
}

type FilesystemNotifierConfiguration struct {
	Directory string `mapstructure:"directory" validate:"required"`
}

type ActivityConfiguration struct {
	Type       string                           `mapstructure:"type"       validate:"required,oneof=loki filesystem"`
	Loki       *LokiConfiguration               `mapstructure:"loki"       validate:"required_if=Type loki"`
	Filesystem *FilesystemActivityConfiguration `mapstructure:"filesystem" validate:"required_if=Type filesystem"`
}

type FilesystemActivityConfiguration struct {
	Directory string `mapstructure:"directory" validate:"required"`
}

type LokiConfiguration struct {
	Endpoint string `mapstructure:"endpoint" validate:"required,http_url"`
}

type StaticConfiguration struct {
	Enabled   bool   `mapstructure:"enabled"`
	Directory string `mapstructure:"directory"`
}

// AuthConfig groups authentication-related configuration for services.
// This reduces the number of individual fields passed to services and
// makes it easier to add new auth-related config without modifying service structs.
type AuthConfig struct {
	JWTSecret          string
	MFAEncryptionKey   string
	MFARequired        bool
	AccessTokenExpiry  int
	RefreshTokenExpiry int
	MFATokenExpiry     int
	WebURL             string
}

// GetAuthConfig extracts authentication configuration from AppConfiguration.
func (c *AppConfiguration) GetAuthConfig() AuthConfig {
	return AuthConfig{
		JWTSecret:          c.JWTSecret,
		MFAEncryptionKey:   c.MFAEncryptionKey,
		MFARequired:        c.MFARequired,
		AccessTokenExpiry:  c.AccessTokenExpiry,
		RefreshTokenExpiry: c.RefreshTokenExpiry,
		MFATokenExpiry:     c.MFATokenExpiry,
		WebURL:             c.WebURL,
	}
}
