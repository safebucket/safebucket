package helpers

import (
	"context"
	"fmt"
	"github.com/redis/rueidis"
	"time"
)

func RegisterPlatform(id string, client rueidis.Client) error {
	ctx := context.Background()
	sortedSetKey := "platform:identity"
	currentTime := float64(time.Now().Unix())
	err := client.Do(ctx, client.B().Zadd().Key(sortedSetKey).ScoreMember().ScoreMember(currentTime, id).Build()).Error()
	return err
}

func DeleteInactivePlatform(client rueidis.Client) error {
	ctx := context.Background()
	sortedSetKey := "platform:identity"
	currentTime := float64(time.Now().Unix())
	maxLifetime := float64(60 * 2) // 2 mn
	err := client.Do(ctx, client.B().Zremrangebyscore().Key(sortedSetKey).Min("-inf").Max(fmt.Sprintf("%f", currentTime-maxLifetime)).Build()).Error()
	return err
}

func StartIdentityTicker(id string, client rueidis.Client) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := RegisterPlatform(id, client)
			if err != nil {
				return err
			}
			err = DeleteInactivePlatform(client)
			if err != nil {
				return err
			}
		}
	}
}
