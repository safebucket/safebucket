package workers

import (
	"context"
	"path"
	"time"

	"api/internal/models"
	"api/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	GCStaleUploadThreshold = 20 * time.Minute
	GCBatchSize            = 100
)

// GarbageCollectorWorker periodically cleans up orphaned database records.
type GarbageCollectorWorker struct {
	DB          *gorm.DB
	Storage     storage.IStorage
	RunInterval time.Duration
}

func (w *GarbageCollectorWorker) Start(ctx context.Context) {
	StartPeriodicWorker(ctx, "garbage_collector", w.RunInterval, []WorkerTask{
		{Name: "stale_uploads", Fn: w.cleanupStaleUploads},
		{Name: "expired_challenges", Fn: w.cleanupExpiredChallenges},
		{Name: "expired_files", Fn: w.cleanupExpiredFiles},
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

// cleanupExpiredFiles hard-deletes files that have passed their expiration date.
func (w *GarbageCollectorWorker) cleanupExpiredFiles(ctx context.Context) (int, error) {
	var files []models.File

	if err := w.DB.Unscoped().
		Where("expires_at IS NOT NULL AND expires_at < ? AND status = ?", time.Now(), models.FileStatusUploaded).
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
		zap.L().Warn("Failed to delete expired files from storage", zap.Error(err))
	}

	result := w.DB.Unscoped().Delete(&models.File{}, fileIDs)
	if result.Error != nil {
		return 0, result.Error
	}

	if result.RowsAffected > 0 {
		zap.L().Debug("Deleted expired files", zap.Int64("count", result.RowsAffected))
	}

	return int(result.RowsAffected), nil
}
