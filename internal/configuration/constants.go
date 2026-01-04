package configuration

const AppName = "safebucket"

const (
	CacheMaxAppIdentityLifetime = 60
	CacheAppIdentityKey         = "app:identity"
	CacheAppRateLimitKey        = "app:ratelimit:%s"
	CacheMFAAttemptsKey         = "mfa:attempts:%s"
	CacheTOTPUsedKey            = "totp:used:%s:%s"
)

const (
	EventsNotifications  = "notifications"
	EventsObjectDeletion = "object_deletion"
	EventsBucketEvents   = "bucket_events"
)

const UploadPolicyExpirationInMinutes = 15

const (
	SecurityChallengeExpirationMinutes = 30
	SecurityChallengeMaxFailedAttempts = 3
)

const (
	// MaxMFADevicesPerUser is the maximum number of MFA devices allowed per user.
	MaxMFADevicesPerUser = 5
	// TOTPCodeTTL is the time-to-live for TOTP code replay protection (in seconds).
	TOTPCodeTTL = 90
	// MFAMaxAttempts is the maximum number of failed MFA verification attempts before lockout.
	MFAMaxAttempts = 5
	// MFALockoutSeconds is the lockout duration after max failed MFA attempts (in seconds).
	MFALockoutSeconds = 900
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
