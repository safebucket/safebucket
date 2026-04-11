package workers

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	GCStaleUploadThreshold = 20 * time.Minute
	GCBatchSize            = 100
)

type GarbageCollectorWorker struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	Cache              cache.ICache
	ActivityLogger     activity.IActivityLogger
	RunInterval        time.Duration
	RefreshTokenExpiry int
}

func (w *GarbageCollectorWorker) Start(ctx context.Context) {
	StartPeriodicWorker(ctx, "garbage_collector", w.RunInterval, []WorkerTask{
		{Name: "stale_uploads", Fn: w.cleanupStaleUploads},
		{Name: "expired_challenges", Fn: w.cleanupExpiredChallenges},
		{Name: "expired_files", Fn: w.cleanupExpiredFiles},
		{Name: "expired_shares", Fn: w.cleanupExpiredShares},
		{Name: "max_views_shares", Fn: w.cleanupMaxViewsShares},
		{Name: "expired_sessions", Fn: w.cleanupExpiredSessions},
	})
}

// cleanupStaleUploads deletes files stuck in "uploading" status beyond the threshold.
func (w *GarbageCollectorWorker) cleanupStaleUploads(_ context.Context) (int, error) {
	threshold := time.Now().Add(-GCStaleUploadThreshold)

	result := w.DB.Unscoped().
		Where("status = ? AND created_at < ?", models.FileStatusUploading, threshold).
		Limit(GCBatchSize).
		Delete(&models.File{})

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		zap.L().Debug("Deleted stale uploading files", zap.Int64("count", result.RowsAffected))
	}

	return int(result.RowsAffected), nil
}

// cleanupExpiredChallenges hard-deletes challenges that have expired.
func (w *GarbageCollectorWorker) cleanupExpiredChallenges(_ context.Context) (int, error) {
	now := time.Now()

	result := w.DB.Unscoped().
		Where("(expires_at IS NOT NULL AND expires_at < ?) OR deleted_at IS NOT NULL", now).
		Limit(GCBatchSize).
		Delete(&models.Challenge{})

	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		zap.L().Debug("Deleted expired challenges", zap.Int64("count", result.RowsAffected))
	}

	return int(result.RowsAffected), nil
}

// cleanupExpiredFiles hard-deletes files that have passed their expiration date.
func (w *GarbageCollectorWorker) cleanupExpiredFiles(_ context.Context) (int, error) {
	var files []models.File

	if err := w.DB.Unscoped().
		Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).
		Limit(GCBatchSize).
		Find(&files).Error; err != nil {
		return 0, err
	}

	if len(files) == 0 {
		return 0, nil
	}

	storagePaths := make([]string, len(files))
	fileIDs := make([]uuid.UUID, len(files))
	for i, file := range files {
		storagePaths[i] = path.Join("buckets", file.BucketID.String(), file.ID.String())
		fileIDs[i] = file.ID
	}

	if err := w.Storage.RemoveObjects(storagePaths); err != nil {
		return 0, fmt.Errorf("failed to remove objects from storage: %w", err)
	}

	var rowsAffected int64

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Delete(&models.File{}, fileIDs)
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected

		for _, file := range files {
			action := models.Activity{
				Message: activity.FileExpired,
				Object:  file.ToActivity(),
				Filter: activity.NewLogFilter(map[string]string{
					"action":      rbac.ActionDelete.String(),
					"bucket_id":   file.BucketID.String(),
					"file_id":     file.ID.String(),
					"object_type": rbac.ResourceFile.String(),
				}),
			}
			if err := w.ActivityLogger.Send(action); err != nil {
				zap.L().Error("Failed to log file expiration activity", zap.Error(err))
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		zap.L().Debug("Deleted expired files", zap.Int64("count", rowsAffected))
	}

	return int(rowsAffected), nil
}

func (w *GarbageCollectorWorker) cleanupExpiredShares(_ context.Context) (int, error) {
	return w.cleanupShares("expires_at IS NOT NULL AND expires_at < ?", []any{time.Now()}, activity.ShareExpired)
}

func (w *GarbageCollectorWorker) cleanupMaxViewsShares(_ context.Context) (int, error) {
	return w.cleanupShares(
		"max_views IS NOT NULL AND current_views >= max_views",
		nil,
		activity.ShareMaxViewsReached,
	)
}

func (w *GarbageCollectorWorker) cleanupShares(whereClause string, args []any, activityMsg string) (int, error) {
	var shares []models.Share

	if err := w.DB.Unscoped().
		Where(whereClause, args...).
		Limit(GCBatchSize).
		Find(&shares).Error; err != nil {
		return 0, err
	}

	if len(shares) == 0 {
		return 0, nil
	}

	shareIDs := make([]uuid.UUID, len(shares))
	for i, share := range shares {
		shareIDs[i] = share.ID
	}

	var rowsAffected int64

	err := w.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("share_id IN ?", shareIDs).Delete(&models.ShareFile{}).Error; err != nil {
			return err
		}

		result := tx.Unscoped().Delete(&models.Share{}, shareIDs)
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected

		for _, share := range shares {
			action := models.Activity{
				Message: activityMsg,
				Object:  share.ToActivity(),
				Filter: activity.NewLogFilter(map[string]string{
					"action":      rbac.ActionDelete.String(),
					"bucket_id":   share.BucketID.String(),
					"share_id":    share.ID.String(),
					"object_type": rbac.ResourceShare.String(),
				}),
			}
			if err := w.ActivityLogger.Send(action); err != nil {
				zap.L().Error("Failed to log share cleanup activity", zap.Error(err))
			}
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		zap.L().Debug("Deleted shares", zap.String("reason", activityMsg), zap.Int64("count", rowsAffected))
	}

	return int(rowsAffected), nil
}

// cleanupExpiredSessions scans all session sorted sets and removes expired entries.
func (w *GarbageCollectorWorker) cleanupExpiredSessions(_ context.Context) (int, error) {
	keys, err := w.Cache.ScanKeys(
		fmt.Sprintf(configuration.CacheUserSessionsKey, "*"),
		GCBatchSize,
		0,
	)
	if err != nil {
		return 0, err
	}

	cutoff := strconv.FormatInt(
		time.Now().Add(-time.Duration(w.RefreshTokenExpiry)*time.Minute).Unix(),
		10,
	)

	cleaned := 0
	for _, key := range keys {
		if remErr := w.Cache.ZRemRangeByScore(key, "-inf", cutoff); remErr != nil {
			zap.L().Error("Failed to clean session key", zap.String("key", key), zap.Error(remErr))
			continue
		}

		remaining, rangeErr := w.Cache.ZRangeByScoreWithScores(key, "-inf", "+inf")
		if rangeErr != nil {
			continue
		}
		if len(remaining) == 0 {
			_ = w.Cache.Del(key)
		}

		cleaned++
	}

	if cleaned > 0 {
		zap.L().Debug("Cleaned expired sessions", zap.Int("keys_processed", cleaned))
	}

	return cleaned, nil
}
