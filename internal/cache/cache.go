package cache

import (
	"api/internal/models"
	"github.com/redis/rueidis"
	"go.uber.org/zap"
)

type Cache struct {
	c rueidis.Client
}

func InitCache(config models.RedisConfiguration) Cache {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: config.Hosts,
		Password:    config.Password,
	})
	if err != nil {
		zap.L().Error("Failed to connect to redis", zap.Error(err))
	}
	return Cache{c: client}
}
