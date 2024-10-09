package cache

import (
	c "api/internal/configuration"
	"context"
	"fmt"
	"time"
)

func (cache Cache) RegisterPlatform(id string) error {
	ctx := context.Background()
	sortedSetKey := c.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	err := cache.c.Do(ctx, cache.c.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).Error()
	return err
}

func (cache Cache) DeleteInactivePlatform() error {
	ctx := context.Background()
	sortedSetKey := c.CacheAppIdentityKey
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(c.CacheMaxAppIdentityLifetime)
	err := cache.c.Do(ctx, cache.c.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).Error()
	return err
}

func (cache Cache) StartIdentityTicker(id string) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		err := cache.RegisterPlatform(id)
		if err != nil {
			return err
		}
		err = cache.DeleteInactivePlatform()
		if err != nil {
			return err
		}
	}
	return nil
}
