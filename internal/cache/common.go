package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"api/internal/configuration"

	"github.com/redis/rueidis"
	"go.uber.org/zap"
)

type RueidisCache struct {
	client rueidis.Client
}

func newRueidisCache(
	hosts []string,
	password string,
	tlsEnabled bool,
	tlsServerName,
	errorContext string,
) (*RueidisCache, error) {
	clientOption := rueidis.ClientOption{
		InitAddress: hosts,
		Password:    password,
	}

	if tlsEnabled {
		clientOption.TLSConfig = &tls.Config{
			ServerName: tlsServerName,
			MinVersion: tls.VersionTLS12,
		}
	}

	client, err := rueidis.NewClient(clientOption)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", errorContext, err)
	}
	return &RueidisCache{client: client}, nil
}

func (r *RueidisCache) RegisterPlatform(id string) error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	err := r.client.Do(ctx, r.client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).
		Error()
	return err
}

func (r *RueidisCache) DeleteInactivePlatform() error {
	ctx := context.Background()
	sortedSetKey := configuration.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(configuration.CacheMaxAppIdentityLifetime)
	err := r.client.Do(ctx, r.client.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).
		Error()
	return err
}

func (r *RueidisCache) StartIdentityTicker(id string) {
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

func (r *RueidisCache) GetRateLimit(userIdentifier string, requestsPerMinute int) (int, error) {
	ctx := context.Background()

	key := fmt.Sprintf(configuration.CacheAppRateLimitKey, userIdentifier)
	count, err := r.client.Do(ctx, r.client.B().Incr().Key(key).Build()).AsInt64()
	if err != nil {
		return 0, err
	}

	if count == 1 {
		expireErr := r.client.Do(ctx, r.client.B().Expire().Key(key).Seconds(int64(1*time.Minute.Seconds())).Build()).
			Error()
		if expireErr != nil {
			return 0, expireErr
		}
	}

	if int(count) > requestsPerMinute {
		retryAfter, ttlErr := r.client.Do(ctx, r.client.B().Ttl().Key(key).Build()).AsInt64()
		if ttlErr != nil {
			return 0, ttlErr
		}

		return int(retryAfter), nil
	}

	return 0, nil
}

func (r *RueidisCache) Close() error {
	r.client.Close()
	return nil
}

func (r *RueidisCache) IsTOTPCodeUsed(deviceID string, code string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheTOTPUsedKey, deviceID, code)

	result, err := r.client.Do(ctx, r.client.B().Exists().Key(key).Build()).AsInt64()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (r *RueidisCache) MarkTOTPCodeUsed(deviceID string, code string) error {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheTOTPUsedKey, deviceID, code)

	err := r.client.Do(
		ctx,
		r.client.B().Setex().Key(key).Seconds(int64(configuration.TOTPCodeTTL)).Value("1").Build(),
	).Error()
	return err
}

func (r *RueidisCache) GetMFAAttempts(userID string) (int, error) {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)

	count, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).AsInt64()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return 0, nil
		}
		return 0, err
	}
	return int(count), nil
}

func (r *RueidisCache) IncrementMFAAttempts(userID string) error {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)

	_, err := r.client.Do(ctx, r.client.B().Incr().Key(key).Build()).AsInt64()
	if err != nil {
		return err
	}

	err = r.client.Do(
		ctx,
		r.client.B().Expire().Key(key).Seconds(int64(configuration.MFALockoutSeconds)).Build(),
	).Error()
	return err
}

func (r *RueidisCache) ResetMFAAttempts(userID string) error {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)

	return r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error()
}
