package configuration

const (
	CacheMaxAppIdentityLifetime = 60
	CacheAppIdentityKey         = "app:identity"
	CacheAppRateLimitKey        = "app:ratelimit:%s"
)

const (
	EventsNotifications  = "notifications"
	EventsObjectDeletion = "object_deletion"
	EventsBucketEvents   = "bucket_events"
)

const DefaultDomain = "04db8656-d4f6-4f27-a2bd-8fab66155b21"

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
