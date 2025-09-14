package cache

import (
	"api/internal/configuration"
	"api/internal/models"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/rueidis"
	"go.uber.org/zap"
)

type RedisCache struct {
	client rueidis.Client
}

func NewRedisCache(config models.RedisCacheConfiguration) (*RedisCache, error) {
	clientOption := rueidis.ClientOption{
		InitAddress: config.Hosts,
		Password:    config.Password,
	}

	if config.TLSEnabled {
		clientOption.TLSConfig = &tls.Config{
			ServerName: config.TLSServerName,
			MinVersion: tls.VersionTLS12,
		}
	}

	client, err := rueidis.NewClient(clientOption)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}
	return &RedisCache{client: client}, nil
}

func (r *RedisCache) RegisterPlatform(id string) error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	err := r.client.Do(ctx, r.client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).Error()
	return err
}

func (r *RedisCache) DeleteInactivePlatform() error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(configuration.CacheMaxAppIdentityLifetime)
	err := r.client.Do(ctx, r.client.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).Error()
	return err
}

func (r *RedisCache) StartIdentityTicker(id string) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		err := r.RegisterPlatform(id)
		if err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
		err = r.DeleteInactivePlatform()
		if err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
	}
}

func (r *RedisCache) GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error) {
	ctx := context.Background()

	key := fmt.Sprintf(configuration.CacheAppRateLimitKey, userIdentifier)
	count, err := r.client.Do(ctx, r.client.B().Incr().Key(key).Build()).AsInt64()

	if err != nil {
		return 0, err
	}

	if count == 1 {
		err := r.client.Do(ctx, r.client.B().Expire().Key(key).Seconds(int64(1*time.Minute.Seconds())).Build()).Error()
		if err != nil {
			return 0, err
		}
	}

	if int(count) > requestsPerMinute {
		retryAfter, err := r.client.Do(ctx, r.client.B().Ttl().Key(key).Build()).AsInt64()

		if err != nil {
			return 0, err
		}

		return int(retryAfter), nil
	}

	return 0, nil
}

func (v *RedisCache) Close() error {
	v.client.Close()
	return nil
}
