package cache

import (
	"github.com/safebucket/safebucket/internal/models"
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
