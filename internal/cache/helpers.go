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

type ActiveSession struct {
	SID       string
	CreatedAt time.Time
}

func sessionKey(userID string) string {
	return fmt.Sprintf(configuration.CacheUserSessionsKey, userID)
}

func sessionCutoff(maxAge time.Duration) string {
	return strconv.FormatFloat(float64(time.Now().Add(-maxAge).Unix()), 'f', 0, 64)
}

func CreateSession(c ICache, userID string, sid string) error {
	return c.ZAdd(sessionKey(userID), float64(time.Now().Unix()), sid)
}

func IsSessionActive(c ICache, userID string, sid string, maxAge time.Duration) (bool, error) {
	score, err := c.ZScore(sessionKey(userID), sid)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			return false, nil
		}
		return false, err
	}
	cutoff := float64(time.Now().Add(-maxAge).Unix())
	return score >= cutoff, nil
}

func ListActiveSessions(c ICache, userID string, maxAge time.Duration) ([]ActiveSession, error) {
	key := sessionKey(userID)
	cutoff := sessionCutoff(maxAge)

	_ = c.ZRemRangeByScore(key, "-inf", cutoff)

	entries, err := c.ZRangeByScoreWithScores(key, cutoff, "+inf")
	if err != nil {
		return nil, err
	}

	sessions := make([]ActiveSession, 0, len(entries))
	for _, e := range entries {
		sessions = append(sessions, ActiveSession{
			SID:       e.Member,
			CreatedAt: time.Unix(int64(e.Score), 0),
		})
	}
	return sessions, nil
}

func RevokeSession(c ICache, userID string, sid string) error {
	return c.ZAdd(sessionKey(userID), 0, sid)
}

func RevokeOtherSessions(c ICache, userID string, currentSID string, maxAge time.Duration) error {
	key := sessionKey(userID)
	cutoff := sessionCutoff(maxAge)

	entries, err := c.ZRangeByScoreWithScores(key, cutoff, "+inf")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Member != currentSID {
			if zErr := c.ZAdd(key, 0, e.Member); zErr != nil {
				return zErr
			}
		}
	}
	return nil
}

func RevokeAllSessions(c ICache, userID string) error {
	return c.Del(sessionKey(userID))
}
