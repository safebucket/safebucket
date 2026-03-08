package cache

import (
	"github.com/safebucket/safebucket/internal/models"
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
