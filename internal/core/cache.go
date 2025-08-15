package core

import (
	"api/internal/cache"
	"api/internal/models"
	"go.uber.org/zap"
)

func NewCache(config models.CacheConfiguration) cache.ICache {
	switch config.Type {
	case "redis":
		if config.Redis == nil {
			zap.L().Fatal("Redis configuration is required when cache type is redis")
		}
		cacheInstance, err := cache.NewRedisCache(*config.Redis)
		if err != nil {
			zap.L().Fatal("Failed to initialize Redis cache", zap.Error(err))
		}
		return cacheInstance
	case "valkey":
		if config.Valkey == nil {
			zap.L().Fatal("Valkey configuration is required when cache type is valkey")
		}
		cacheInstance, err := cache.NewValkeyCache(*config.Valkey)
		if err != nil {
			zap.L().Fatal("Failed to initialize Valkey cache", zap.Error(err))
		}
		return cacheInstance
	default:
		zap.L().Fatal("Unsupported cache type", zap.String("type", config.Type))
		return nil
	}
}
