package cache

type ICache interface {
	RegisterPlatform(id string) error
	DeleteInactivePlatform() error
	StartIdentityTicker(id string)

	GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error)

	Close() error
}
