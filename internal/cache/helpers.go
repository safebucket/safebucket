package cache

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
)

func GetMFAAttempts(c ICache, userID string) (int, error) {
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)
	val, err := c.Get(key)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return 0, nil
		}
		return 0, err
	}
	count, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func IncrementMFAAttempts(c ICache, userID string) error {
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)
	_, err := c.Incr(key)
	if err != nil {
		return err
	}
	return c.Expire(key, time.Duration(configuration.MFALockoutSeconds)*time.Second)
}

func ResetMFAAttempts(c ICache, userID string) error {
	key := fmt.Sprintf(configuration.CacheMFAAttemptsKey, userID)
	return c.Del(key)
}

func MarkTOTPCodeUsed(c ICache, deviceID string, code string) (bool, error) {
	key := fmt.Sprintf(configuration.CacheTOTPUsedKey, deviceID, code)
	return c.SetNX(key, "1", time.Duration(configuration.TOTPCodeTTL)*time.Second)
}

func GetRateLimit(c ICache, userIdentifier string, requestsPerMinute int) (int, error) {
	key := fmt.Sprintf(configuration.CacheAppRateLimitKey, userIdentifier)

	count, err := c.Incr(key)
	if err != nil {
		return 0, err
	}

	if count == 1 {
		if expErr := c.Expire(key, 1*time.Minute); expErr != nil {
			return 0, expErr
		}
	}

	if int(count) > requestsPerMinute {
		ttl, ttlErr := c.TTL(key)
		if ttlErr != nil {
			return 0, ttlErr
		}
		return int(ttl.Seconds()), nil
	}

	return 0, nil
}

func RefreshLock(c ICache, key string, instanceID string, ttl time.Duration) (bool, error) {
	current, err := c.Get(key)
	if err == nil && current == instanceID {
		if expErr := c.Expire(key, ttl); expErr != nil {
			return false, expErr
		}
		return true, nil
	}
	return false, err
}
