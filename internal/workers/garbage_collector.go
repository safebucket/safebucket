package workers

import (
	"context"
	"time"

	"api/internal/activity"
	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	GCStaleUploadThreshold = 20 * time.Minute
	GCBatchSize            = 100
)

// GarbageCollectorWorker periodically cleans up orphaned database records.
type GarbageCollectorWorker struct {
	DB             *gorm.DB
	RunInterval    time.Duration
	ActivityLogger activity.IActivityLogger
}

func (w *GarbageCollectorWorker) Start(ctx context.Context) {
	tracker := &RunTracker{DB: w.DB, ActivityLogger: w.ActivityLogger}
	StartPeriodicWorker(ctx, tracker, "garbage_collector", w.RunInterval, []WorkerTask{
		{Name: "stale_uploads", Fn: w.cleanupStaleUploads},
		{Name: "expired_challenges", Fn: w.cleanupExpiredChallenges},
	})
}

// cleanupStaleUploads deletes files stuck in "uploading" status beyond the threshold.
func (w *GarbageCollectorWorker) cleanupStaleUploads(ctx context.Context) (int, error) {
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

// cleanupExpiredChallenges hard-deletes challenges that have expired
func (w *GarbageCollectorWorker) cleanupExpiredChallenges(ctx context.Context) (int, error) {
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
