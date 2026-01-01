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

// TryAcquireLock attempts to acquire a distributed lock using SET NX EX.
// Returns true if lock was acquired, false if already held by another instance.
func (r *RueidisCache) TryAcquireLock(key string, instanceID string, ttlSeconds int) (bool, error) {
	ctx := context.Background()
	// SET key value NX EX ttl - atomic set-if-not-exists with expiration
	result, err := r.client.Do(ctx,
		r.client.B().Set().Key(key).Value(instanceID).Nx().Ex(time.Duration(ttlSeconds)*time.Second).Build(),
	).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Key already exists, lock not acquired
			return false, nil
		}
		return false, err
	}
	return result == "OK", nil
}

// RefreshLock extends the TTL of an existing lock if held by this instance.
// Returns true if refresh succeeded, false if lock is no longer held.
func (r *RueidisCache) RefreshLock(key string, instanceID string, ttlSeconds int) (bool, error) {
	ctx := context.Background()
	// GET to verify we still hold the lock
	current, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			// Lock no longer exists
			return false, nil
		}
		return false, err
	}
	if current != instanceID {
		// Lock held by another instance
		return false, nil
	}
	// EXPIRE to refresh TTL
	err = r.client.Do(ctx,
		r.client.B().Expire().Key(key).Seconds(int64(ttlSeconds)).Build(),
	).Error()
	return err == nil, err
}

// ReleaseLock releases the lock if held by this instance.
// Uses Lua script for atomic check-and-delete.
func (r *RueidisCache) ReleaseLock(key string, instanceID string) error {
	ctx := context.Background()
	// Only delete if we hold the lock (Lua script for atomicity)
	script := `if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`
	_, err := r.client.Do(ctx,
		r.client.B().Eval().Script(script).Numkeys(1).Key(key).Arg(instanceID).Build(),
	).ToInt64()
	return err
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

func (r *RueidisCache) MarkTOTPCodeUsed(deviceID string, code string) (bool, error) {
	ctx := context.Background()
	key := fmt.Sprintf(configuration.CacheTOTPUsedKey, deviceID, code)

	// SET key value NX EX ttl
	// Returns OK if set, nil (RedisNil) if not set (already exists)
	err := r.client.Do(
		ctx,
		r.client.B().Set().Key(key).Value("1").Nx().ExSeconds(int64(configuration.TOTPCodeTTL)).Build(),
	).Error()

	if err != nil {
		if rueidis.IsRedisNil(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
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
