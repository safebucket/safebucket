package cache

import (
	"api/internal/models"
)

type RedisCache = RueidisCache

func NewRedisCache(config models.RedisCacheConfiguration) (*RedisCache, error) {
	return newRueidisCache(
		config.Hosts,
		config.Password,
		config.TLSEnabled,
		config.TLSServerName,
		"redis",
	)
}
