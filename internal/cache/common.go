package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/rueidis"
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

func (r *RueidisCache) Get(key string) (string, error) {
	ctx := context.Background()
	val, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return "", ErrKeyNotFound
		}
		return "", err
	}
	return val, nil
}

func (r *RueidisCache) SetNX(key string, value string, ttl time.Duration) (bool, error) {
	ctx := context.Background()
	err := r.client.Do(ctx,
		r.client.B().Set().Key(key).Value(value).Nx().Ex(ttl).Build(),
	).Error()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RueidisCache) Del(key string) error {
	ctx := context.Background()
	return r.client.Do(ctx, r.client.B().Del().Key(key).Build()).Error()
}

func (r *RueidisCache) Incr(key string) (int64, error) {
	ctx := context.Background()
	return r.client.Do(ctx, r.client.B().Incr().Key(key).Build()).AsInt64()
}

func (r *RueidisCache) Expire(key string, ttl time.Duration) error {
	ctx := context.Background()
	return r.client.Do(ctx, r.client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()).Error()
}

func (r *RueidisCache) TTL(key string) (time.Duration, error) {
	ctx := context.Background()
	seconds, err := r.client.Do(ctx, r.client.B().Ttl().Key(key).Build()).AsInt64()
	if err != nil {
		return 0, err
	}
	return time.Duration(seconds) * time.Second, nil
}

func (r *RueidisCache) ZAdd(key string, score float64, member string) error {
	ctx := context.Background()
	return r.client.Do(ctx,
		r.client.B().Zadd().Key(key).ScoreMember().ScoreMember(score, member).Build(),
	).Error()
}

func (r *RueidisCache) ZScore(key string, member string) (float64, error) {
	ctx := context.Background()
	score, err := r.client.Do(ctx, r.client.B().Zscore().Key(key).Member(member).Build()).AsFloat64()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return 0, ErrKeyNotFound
		}
		return 0, err
	}
	return score, nil
}

func (r *RueidisCache) ZRangeByScoreWithScores(
	key string, minScore string, maxScore string,
) ([]ZScoreEntry, error) {
	ctx := context.Background()
	scores, err := r.client.Do(ctx,
		r.client.B().Zrange().Key(key).Min(minScore).Max(maxScore).Byscore().Withscores().Build(),
	).AsZScores()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return nil, nil
		}
		return nil, err
	}
	entries := make([]ZScoreEntry, len(scores))
	for i, s := range scores {
		entries[i] = ZScoreEntry{Member: s.Member, Score: s.Score}
	}
	return entries, nil
}

func (r *RueidisCache) ZRemRangeByScore(key string, minScore string, maxScore string) error {
	ctx := context.Background()
	return r.client.Do(ctx,
		r.client.B().Zremrangebyscore().Key(key).Min(minScore).Max(maxScore).Build(),
	).Error()
}

func (r *RueidisCache) ScanKeys(pattern string, count int64, limit int64) ([]string, error) {
	ctx := context.Background()
	var keys []string
	scanner := rueidis.NewScanner(func(cursor uint64) (rueidis.ScanEntry, error) {
		return r.client.Do(ctx,
			r.client.B().Scan().Cursor(cursor).Match(pattern).Count(count).Build(),
		).AsScanEntry()
	})
	for key := range scanner.Iter() {
		keys = append(keys, key)
		if limit > 0 && int64(len(keys)) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *RueidisCache) Close() {
	r.client.Close()
}
