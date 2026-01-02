package cache

type ICache interface {
	RegisterPlatform(id string) error
	DeleteInactivePlatform() error
	StartIdentityTicker(id string)

	GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error)

	// TryAcquireLock attempts to acquire a distributed lock. Returns true if acquired.
	TryAcquireLock(key string, instanceID string, ttlSeconds int) (bool, error)

	// RefreshLock extends the TTL of an existing lock if held by this instance.
	RefreshLock(key string, instanceID string, ttlSeconds int) (bool, error)

	Close() error
}
