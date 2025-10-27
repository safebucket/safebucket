package cache

import (
	"api/internal/models"
)

type ValkeyCache = RueidisCache

func NewValkeyCache(cacheConfig models.ValkeyCacheConfiguration) (*ValkeyCache, error) {
	return newRueidisCache(
		cacheConfig.Hosts,
		cacheConfig.Password,
		cacheConfig.TLSEnabled,
		cacheConfig.TLSServerName,
		"valkey",
	)
}
