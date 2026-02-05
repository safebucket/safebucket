package workers

import (
	"context"
	"path"
	"time"

	"api/internal/activity"
	"api/internal/events"
	"api/internal/messaging"
	"api/internal/models"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// FileBatchSize is the number of files to process in each cleanup batch.
	FileBatchSize = 100
	// FolderBatchSize is the number of folders to process in each cleanup batch.
	FolderBatchSize = 50
)

// watermillPublisherAdapter adapts messaging.IPublisher to message.Publisher interface.
// This allows TrashExpiration.Trigger() to work with our EventRouter.
type watermillPublisherAdapter struct {
	publisher messaging.IPublisher
}

func (a *watermillPublisherAdapter) Publish(_ string, msgs ...*message.Message) error {
	return a.publisher.Publish(msgs...)
}

func (a *watermillPublisherAdapter) Close() error {
	return a.publisher.Close()
}

// TrashCleanupWorker handles application-level trash cleanup for storage providers
// that don't support bucket lifecycle policies (e.g., Storj).
// It queries for expired trashed items and triggers existing events to handle cleanup.
type TrashCleanupWorker struct {
	DB                 *gorm.DB
	Publisher          messaging.IPublisher
	TrashRetentionDays int
	RunInterval        time.Duration
	ActivityLogger     activity.IActivityLogger
}

// Start begins the trash cleanup worker loop.
// It runs an immediate cleanup on startup, then runs on the configured interval.
// The worker respects context cancellation for graceful shutdown.
func (w *TrashCleanupWorker) Start(ctx context.Context) {
	zap.L().Info("Starting trash cleanup worker",
		zap.Int("retention_days", w.TrashRetentionDays),
		zap.Duration("interval", w.RunInterval))

	w.runCleanup(ctx)

	ticker := time.NewTicker(w.RunInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Trash cleanup worker shutting down")
			return
		case <-ticker.C:
			w.runCleanup(ctx)
		}
	}
}

// runCleanup performs a full cleanup cycle for expired files and folders.
func (w *TrashCleanupWorker) runCleanup(ctx context.Context) {
	startTime := time.Now()
	zap.L().Info("Starting trash cleanup cycle")

	tracker := &RunTracker{
		DB:             w.DB,
		ActivityLogger: w.ActivityLogger,
	}

	run, err := tracker.StartRun("trash_cleanup")
	if err != nil {
		zap.L().Error("Failed to start worker run tracking", zap.Error(err))
		return
	}

	var runFailed bool

	filesQueued, err := w.cleanupExpiredFiles(ctx)
	if err != nil {
		zap.L().Error("Failed to cleanup expired files", zap.Error(err))
		runFailed = true
	}

	foldersQueued, err := w.cleanupExpiredFolders(ctx)
	if err != nil {
		zap.L().Error("Failed to cleanup expired folders", zap.Error(err))
		runFailed = true
	}

	if runFailed {
		tracker.FailRun(run)
	} else {
		tracker.CompleteRun(run)
	}

	zap.L().Info("Trash cleanup cycle complete",
		zap.Int("files_queued", filesQueued),
		zap.Int("folders_queued", foldersQueued),
		zap.Duration("duration", time.Since(startTime)))
}

// cleanupExpiredFiles finds root-level files (not in any folder) that have been
// in trash longer than the retention period and triggers TrashExpiration events.
// Files inside folders are cleaned up by FolderPurge when the parent folder expires.
func (w *TrashCleanupWorker) cleanupExpiredFiles(ctx context.Context) (int, error) {
	expirationTime := time.Now().AddDate(0, 0, -w.TrashRetentionDays)
	totalQueued := 0

	select {
	case <-ctx.Done():
		return totalQueued, nil
	default:
	}

	var files []models.File
	result := w.DB.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ? AND folder_id IS NULL", expirationTime).
		Limit(FileBatchSize).
		Find(&files)

	if result.Error != nil {
		return 0, result.Error
	}

	if len(files) == 0 {
		return 0, nil
	}

	zap.L().Debug("Processing expired files batch", zap.Int("count", len(files)))

	adapter := &watermillPublisherAdapter{publisher: w.Publisher}

	for _, file := range files {
		markerPath := path.Join("trash", file.BucketID.String(), "files", file.ID.String())
		event := events.NewTrashExpirationFromBucketEvent(file.BucketID, markerPath)
		event.Trigger(adapter)

		zap.L().Debug("Triggered expiration for file",
			zap.String("file_id", file.ID.String()),
			zap.String("file_name", file.Name))

		totalQueued++
	}

	return totalQueued, nil
}

// cleanupExpiredFolders finds root-level trashed folders that have expired
// and triggers FolderPurge events to handle recursive deletion.
func (w *TrashCleanupWorker) cleanupExpiredFolders(ctx context.Context) (int, error) {
	expirationTime := time.Now().AddDate(0, 0, -w.TrashRetentionDays)
	totalQueued := 0

	// Only get root-level trashed folders (no parent)
	// FolderPurge events will handle children recursively
	var folders []models.Folder
	result := w.DB.Unscoped().
		Where("deleted_at IS NOT NULL AND deleted_at < ? AND folder_id IS NULL", expirationTime).
		Limit(FolderBatchSize).
		Find(&folders)

	if result.Error != nil {
		return 0, result.Error
	}

	if len(folders) == 0 {
		return 0, nil
	}

	zap.L().Debug("Processing expired folders", zap.Int("count", len(folders)))

	for _, folder := range folders {
		select {
		case <-ctx.Done():
			return totalQueued, nil
		default:
		}

		event := events.NewFolderPurge(w.Publisher, folder.BucketID, folder.ID, uuid.Nil)
		event.Trigger()

		zap.L().Debug("Triggered purge for expired folder",
			zap.String("folder_id", folder.ID.String()),
			zap.String("folder_name", folder.Name))

		totalQueued++
	}

	if err := w.cleanupOrphanedFolders(ctx, expirationTime); err != nil {
		return totalQueued, err
	}

	return totalQueued, nil
}

// cleanupOrphanedFolders finds and cleans up nested trashed folders
// whose parent folders have already been deleted from the database.
func (w *TrashCleanupWorker) cleanupOrphanedFolders(ctx context.Context, expirationTime time.Time) error {
	var folders []models.Folder
	// Find trashed folders with a folder_id that no longer exists
	result := w.DB.Unscoped().
		Where(`deleted_at IS NOT NULL AND deleted_at < ? AND folder_id IS NOT NULL
			   AND folder_id NOT IN (SELECT id FROM folders WHERE deleted_at IS NULL)`, expirationTime).
		Limit(FolderBatchSize).
		Find(&folders)

	if result.Error != nil {
		return result.Error
	}

	for _, folder := range folders {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		event := events.NewFolderPurge(w.Publisher, folder.BucketID, folder.ID, uuid.Nil)
		event.Trigger()

		zap.L().Debug("Triggered purge for orphaned folder",
			zap.String("folder_id", folder.ID.String()),
			zap.String("folder_name", folder.Name))
	}

	return nil
}
