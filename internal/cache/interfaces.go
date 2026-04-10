package cache

import (
	"errors"
	"time"
)

var ErrKeyNotFound = errors.New("cache: key not found")

// ZScoreEntry represents a sorted set member with its score.
type ZScoreEntry struct {
	Member string
	Score  float64
}

type ICache interface {
	Get(key string) (string, error)
	SetNX(key string, value string, ttl time.Duration) (bool, error)
	Del(key string) error
	Incr(key string) (int64, error)
	Expire(key string, ttl time.Duration) error
	TTL(key string) (time.Duration, error)
	ZAdd(key string, score float64, member string) error
	ZRangeByScoreWithScores(key string, minScore string, maxScore string) ([]ZScoreEntry, error)
	ZScore(key string, member string) (float64, error)
	ZRemRangeByScore(key string, minScore string, maxScore string) error
	ScanKeys(pattern string, count int64, limit int64) ([]string, error)
	Close()
}
