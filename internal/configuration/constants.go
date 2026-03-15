package configuration

const AppName = "safebucket"

const (
	AudienceAccessToken  = "app:*"
	AudienceRefreshToken = "auth:refresh"
	AudienceMFALogin     = "auth:mfa:login"
	AudienceMFAReset     = "auth:mfa:password-reset"
)

// JWT Token expiry times (in minutes).
const (
	AccessTokenExpiry  = 60
	RefreshTokenExpiry = 600
	MFATokenExpiry     = 5 // For restricted access during MFA flow
)

const (
	CacheMaxAppIdentityLifetime = 60
	CacheAppIdentityKey         = "app:identity"
	CacheAppRateLimitKey        = "app:ratelimit:%s"
	CacheAppWorkerLockKey       = "app:worker:lock:%s" //nolint:gosec // not a credential
	CacheAppWorkerLockTTL       = 60
	CacheAppWorkerLockRefresh   = 55
	CacheMFAAttemptsKey         = "mfa:attempts:%s"
	CacheTOTPUsedKey            = "totp:used:%s:%s"
	CacheUserSessionsKey        = "user:sessions:%s"
)

const (
	EventsNotifications  = "notifications"
	EventsObjectDeletion = "object_deletion"
	EventsBucketEvents   = "bucket_events"
)

const UploadPolicyExpirationInMinutes = 15

const (
	SecurityChallengeExpirationMinutes = 5
	SecurityChallengeMaxFailedAttempts = 3
)

const (
	MaxMFADevicesPerUser = 5
	TOTPCodeTTL          = 90
	MFAMaxAttempts       = 5
	MFALockoutSeconds    = 900
)

const (
	ProviderPostgres = "postgres"
	ProviderSQLite   = "sqlite"
)

const (
	PostgresMaxOpenConns    = 25
	PostgresMaxIdleConns    = 10
	PostgresConnMaxLifetime = 30 // in minutes
)

const (
	ProviderJetstream = "jetstream"
	ProviderMinio     = "minio"
	ProviderGCP       = "gcp"
	ProviderAWS       = "aws"
	ProviderRustFS    = "rustfs"
	ProviderS3        = "s3"
	ProviderMemory    = "memory"
)

const (
	CacheNotifyBatchCountKey = "notify:batch:count:%s"
	CacheNotifyBatchMetaKey  = "notify:batch:meta:%s"
	CacheNotifyBatchesKey    = "notify:batches"
	CacheNotifyFlush         = 30
	CacheNotifyBatchTTL      = CacheNotifyFlush + 5
)

const BulkActionsLimit = 1000

var ArrayConfigFields = []string{
	"app.trusted_proxies",
	"cors.allowed_origins",
	"cache.redis.hosts",
	"cache.valkey.hosts",
}

var ConfigFileSearchPaths = []string{
	"./config.yaml",
	"templates/config.yaml",
}

var AuthProviderKeys = []string{
	"name",
	"client_id",
	"client_secret",
	"issuer",
}
