package workers

import (
	"context"
	"path"
	"time"

	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FileBatchSize   = 100
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

type TrashCleanupWorker struct {
	DB                 *gorm.DB
	Publisher          messaging.IPublisher
	TrashRetentionDays int
	RunInterval        time.Duration
}

func (w *TrashCleanupWorker) Start(ctx context.Context) {
	StartPeriodicWorker(ctx, "trash_cleanup", w.RunInterval, []WorkerTask{
		{Name: "expired_files", Fn: w.cleanupExpiredFiles},
		{Name: "expired_folders", Fn: w.cleanupExpiredFolders},
	})
}

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

func (w *TrashCleanupWorker) cleanupExpiredFolders(ctx context.Context) (int, error) {
	expirationTime := time.Now().AddDate(0, 0, -w.TrashRetentionDays)
	totalQueued := 0

	// Query root-level trashed folders (no parent)
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
