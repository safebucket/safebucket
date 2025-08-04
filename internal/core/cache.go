package core

import (
	"api/internal/configuration"
	"api/internal/models"
	"context"
	"fmt"
	"github.com/redis/rueidis"
	"go.uber.org/zap"
	"time"
)

type Cache struct {
	client rueidis.Client
}

func InitCache(config models.CacheConfiguration) Cache {
	client, err := rueidis.NewClient(rueidis.ClientOption{
		InitAddress: config.Hosts,
		Password:    config.Password,
	})
	if err != nil {
		zap.L().Error("Failed to connect to redis", zap.Error(err))
	}
	return Cache{client: client}
}

func (c Cache) RegisterPlatform(id string) error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	err := c.client.Do(ctx, c.client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).Error()
	return err
}

func (c Cache) DeleteInactivePlatform() error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(configuration.CacheMaxAppIdentityLifetime)
	err := c.client.Do(ctx, c.client.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).Error()
	return err
}

func (c Cache) StartIdentityTicker(id string) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		err := c.RegisterPlatform(id)
		if err != nil {
			zap.L().Fatal("Platform identity ticker crashed", zap.Error(err))
		}
		err = c.DeleteInactivePlatform()
		if err != nil {
			zap.L().Fatal("Platform identity ticker crashed", zap.Error(err))
		}
	}
}

func (c Cache) GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error) {
	ctx := context.Background()

	key := fmt.Sprintf(configuration.CacheAppRateLimitKey, userIdentifier)
	count, err := c.client.Do(ctx, c.client.B().Incr().Key(key).Build()).AsInt64()

	if err != nil {
		return 0, err
	}

	if count == 1 {
		err := c.client.Do(ctx, c.client.B().Expire().Key(key).Seconds(int64(1*time.Minute.Seconds())).Build()).Error()
		if err != nil {
			return 0, err
		}
	}

	if int(count) > requestsPerMinute {
		retryAfter, err := c.client.Do(ctx, c.client.B().Ttl().Key(key).Build()).AsInt64()

		if err != nil {
			return 0, err
		}

		return int(retryAfter), nil
	}

	return 0, nil
}
