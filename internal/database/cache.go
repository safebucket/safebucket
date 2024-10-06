package database

import (
	"api/internal/models"
	"github.com/redis/rueidis"
	"go.uber.org/zap"
)

func InitCache(config models.RedisConfiguration) rueidis.Client {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: config.Host,
		Password:    config.Password,
	})
	if err != nil {
		zap.L().Error("Failed to connect to redis", zap.Error(err))
	}
	return client
}
