package cache

import (
	"errors"
	"time"
)

var ErrKeyNotFound = errors.New("cache: key not found")

type ICache interface {
	Get(key string) (string, error)
	SetNX(key string, value string, ttl time.Duration) (bool, error)
	Del(key string) error
	Incr(key string) (int64, error)
	Expire(key string, ttl time.Duration) error
	TTL(key string) (time.Duration, error)
	ZAdd(key string, score float64, member string) error
	ZRangeByScore(key string, minScore string, maxScore string) ([]string, error)
	ZRemRangeByScore(key string, minScore string, maxScore string) error
	Close()
}
