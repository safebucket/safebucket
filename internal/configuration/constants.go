package configuration

const CacheMaxAppIdentityLifetime = 60
const CacheAppIdentityKey = "platform:identity"
const CacheAppRateLimitKey = "platform:ratelimit:%s"

const PolicyTableName = "policies"

const NilUUID = "00000000-0000-0000-0000-000000000000"

const DefaultDomain = "04db8656-d4f6-4f27-a2bd-8fab66155b21"

const AuthLocalProviderName = "local"

var ArrayConfigFields = []string{
	"platform.trusted_proxies",
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
