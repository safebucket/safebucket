package configuration

const AppName = "safebucket"

const (
	CacheMaxAppIdentityLifetime = 60
	CacheAppIdentityKey         = "app:identity"
	CacheAppRateLimitKey        = "app:ratelimit:%s"
	CacheAppWorkerLockKey       = "app:worker:lock:%s" //nolint:gosec // not a credential
	CacheAppWorkerLockTTL       = 60
	CacheAppWorkerLockRefresh   = 55
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
