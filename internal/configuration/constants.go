package configuration

const AppName = "safebucket"

// JWT Audience constants for token type separation.
const (
	AudienceAccessToken             = "app:*"
	AudienceRefreshToken            = "auth:refresh"
	AudienceMFALoginToken           = "auth:mfa:login"
	AudienceMFAPasswordResetToken   = "auth:mfa:password-reset"
	AudiencePasswordResetCompletion = "auth:password-reset"
)

// JWT Token expiry times (in minutes).
const (
	AccessTokenExpiry             = 60
	RefreshTokenExpiry            = 600
	MFATokenExpiry                = 5
	PasswordResetMFATokenExpiry   = 5
	PasswordResetCompletionExpiry = 5
)

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
	SecurityChallengeExpirationMinutes = 5
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
