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

type ValkeyCache struct {
	client rueidis.Client
}

func NewValkeyCache(cacheConfig models.ValkeyCacheConfiguration) (*ValkeyCache, error) {

	clientOption := rueidis.ClientOption{
		InitAddress: cacheConfig.Hosts,
		Password:    cacheConfig.Password,
	}

	if cacheConfig.TLSEnabled {
		clientOption.TLSConfig = &tls.Config{
			ServerName: cacheConfig.TLSServerName,
			MinVersion: tls.VersionTLS12,
		}
	}
	client, err := rueidis.NewClient(clientOption)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to valkey: %w", err)
	}
	return &ValkeyCache{client: client}, nil
}

func (v *ValkeyCache) RegisterPlatform(id string) error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	err := v.client.Do(ctx, v.client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).Error()
	return err
}

func (v *ValkeyCache) DeleteInactivePlatform() error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(configuration.CacheMaxAppIdentityLifetime)
	err := v.client.Do(ctx, v.client.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).Error()
	return err
}

func (v *ValkeyCache) StartIdentityTicker(id string) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		err := v.RegisterPlatform(id)
		if err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
		err = v.DeleteInactivePlatform()
		if err != nil {
			zap.L().Fatal("App identity ticker crashed", zap.Error(err))
		}
	}
}

func (v *ValkeyCache) GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error) {
	ctx := context.Background()

	key := fmt.Sprintf(configuration.CacheAppRateLimitKey, userIdentifier)
	count, err := v.client.Do(ctx, v.client.B().Incr().Key(key).Build()).AsInt64()

	if err != nil {
		return 0, err
	}

	if count == 1 {
		err := v.client.Do(ctx, v.client.B().Expire().Key(key).Seconds(int64(1*time.Minute.Seconds())).Build()).Error()
		if err != nil {
			return 0, err
		}
	}

	if int(count) > requestsPerMinute {
		retryAfter, err := v.client.Do(ctx, v.client.B().Ttl().Key(key).Build()).AsInt64()

		if err != nil {
			return 0, err
		}

		return int(retryAfter), nil
	}

	return 0, nil
}

func (v *ValkeyCache) Close() error {
	v.client.Close()
	return nil
}
