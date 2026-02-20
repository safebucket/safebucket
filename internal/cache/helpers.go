package cache

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"api/internal/configuration"
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
