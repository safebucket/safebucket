package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/notifier"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type batchMeta struct {
	RecipientEmail   string             `json:"recipient_email"`
	ActorEmail       string             `json:"actor_email"`
	BucketID         uuid.UUID          `json:"bucket_id"`
	BucketName       string             `json:"bucket_name"`
	NotificationType string             `json:"notification_type"`
	Source           FileActivitySource `json:"source"`
	WebURL           string             `json:"web_url"`
	FirstFileName    string             `json:"first_file_name"`
	ActionText       string             `json:"-"`
}

func batchGroupKey(recipientEmail string, bucketID uuid.UUID, actorEmail string, activityType FileActivityType) string {
	return fmt.Sprintf("%s:%s:%s:%s", recipientEmail, bucketID.String(), actorEmail, string(activityType))
}

func addToBuffer(
	c cache.ICache,
	groupKey string,
	fileName string,
	meta batchMeta,
) (int64, error) {
	countKey := fmt.Sprintf(configuration.CacheNotifyBatchCountKey, groupKey)
	metaKey := fmt.Sprintf(configuration.CacheNotifyBatchMetaKey, groupKey)
	ttl := time.Duration(configuration.CacheNotifyBatchTTL) * time.Second

	count, err := c.Incr(countKey)
	if err != nil {
		return 0, fmt.Errorf("failed to increment batch counter: %w", err)
	}

	if count == 1 {
		meta.FirstFileName = fileName
		metaJSON, _ := json.Marshal(meta)
		if _, err = c.SetNX(metaKey, string(metaJSON), ttl); err != nil {
			return 0, fmt.Errorf("failed to set batch meta key: %w", err)
		}

		if err = c.Expire(countKey, ttl); err != nil {
			return 0, fmt.Errorf("failed to set batch count TTL: %w", err)
		}

		if err = c.ZAdd(configuration.CacheNotifyBatchesKey, float64(time.Now().Unix()), groupKey); err != nil {
			return 0, fmt.Errorf("failed to register batch in sorted set: %w", err)
		}
	}

	return count, nil
}

func flushBuffer(c cache.ICache, n notifier.INotifier, groupKey string) error {
	countKey := fmt.Sprintf(configuration.CacheNotifyBatchCountKey, groupKey)
	metaKey := fmt.Sprintf(configuration.CacheNotifyBatchMetaKey, groupKey)

	countStr, err := c.Get(countKey)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get batch count: %w", err)
	}
	count, _ := strconv.ParseInt(countStr, 10, 64)

	metaStr, err := c.Get(metaKey)
	if err != nil {
		if errors.Is(err, cache.ErrKeyNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get batch meta: %w", err)
	}
	var meta batchMeta
	if err = json.Unmarshal([]byte(metaStr), &meta); err != nil {
		return fmt.Errorf("failed to unmarshal batch meta: %w", err)
	}

	var subject string
	meta.ActionText, subject = composeBatchEmail(meta, count, meta.FirstFileName)

	err = n.NotifyFromTemplate(meta.RecipientEmail, subject, "file_activity", meta)
	if err != nil {
		zap.L().Error("failed to send batched file notification email",
			zap.String("to", meta.RecipientEmail),
			zap.Int64("count", count),
			zap.Error(err))
		return err
	}

	_ = c.Del(countKey)
	_ = c.Del(metaKey)
	_ = c.ZAdd(configuration.CacheNotifyBatchesKey, 0, groupKey)

	return nil
}

func composeBatchEmail(meta batchMeta, count int64, firstName string) (string, string) {
	switch meta.Source {
	case FileActivitySourceShare:
		return composeShareBatchEmail(meta, count)
	case FileActivitySourceUser:
		return composeUserBatchEmail(meta, count, firstName)
	default:
		return composeUserBatchEmail(meta, count, firstName)
	}
}

func composeUserBatchEmail(meta batchMeta, count int64, firstName string) (string, string) {
	var verb, preposition string
	if meta.NotificationType == string(FileActivityUpload) {
		verb = "uploaded"
		preposition = "to"
	} else {
		verb = "downloaded"
		preposition = "from"
	}

	if count == 1 {
		actionText := fmt.Sprintf(
			"%s %s \"%s\" %s bucket \"%s\".",
			meta.ActorEmail, verb, firstName, preposition, meta.BucketName,
		)
		subject := fmt.Sprintf("%s %s a file %s %s",
			meta.ActorEmail, verb, preposition, meta.BucketName)
		return actionText, subject
	}

	actionText := fmt.Sprintf(
		"%s %s %d files %s bucket \"%s\".",
		meta.ActorEmail, verb, count, preposition, meta.BucketName,
	)
	subject := fmt.Sprintf("%s %s %d files %s %s",
		meta.ActorEmail, verb, count, preposition, meta.BucketName)
	return actionText, subject
}

func composeShareBatchEmail(meta batchMeta, count int64) (string, string) {
	var verb, preposition string
	if meta.NotificationType == string(FileActivityUpload) {
		verb = "uploaded"
		preposition = "to"
	} else {
		verb = "downloaded"
		preposition = "from"
	}

	if count == 1 {
		actionText := fmt.Sprintf(
			"A file was %s via sharing link %s bucket \"%s\" (link created by %s).",
			verb, preposition, meta.BucketName, meta.ActorEmail,
		)
		subject := fmt.Sprintf(
			"A file was %s via sharing link %s %s",
			verb, preposition, meta.BucketName,
		)
		return actionText, subject
	}

	actionText := fmt.Sprintf(
		"%d files were %s via sharing link %s bucket \"%s\" (link created by %s).",
		count, verb, preposition, meta.BucketName, meta.ActorEmail,
	)
	subject := fmt.Sprintf(
		"%d files were %s via sharing link %s %s",
		count, verb, preposition, meta.BucketName,
	)
	return actionText, subject
}

// StartFileNotificationBuffer starts a background goroutine that flushes all notification batches.
// The goroutine is registered on wg so callers can wait for it to drain on shutdown.
func StartFileNotificationBuffer(ctx context.Context, wg *sync.WaitGroup, c cache.ICache, n notifier.INotifier) {
	ticker := time.NewTicker(configuration.CacheNotifyFlush * time.Second)

	wg.Go(func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			entries, err := c.ZRangeByScoreWithScores(
				configuration.CacheNotifyBatchesKey,
				"1",
				"+inf",
			)
			if err != nil {
				zap.L().Error("batch flusher: failed to query sorted set", zap.Error(err))
				continue
			}

			for _, entry := range entries {
				if err = flushBuffer(c, n, entry.Member); err != nil {
					zap.L().Error("batch flusher: failed to flush batch",
						zap.String("groupKey", entry.Member), zap.Error(err))
				}
			}

			_ = c.ZRemRangeByScore(configuration.CacheNotifyBatchesKey, "-inf", "0")
		}
	})
}
