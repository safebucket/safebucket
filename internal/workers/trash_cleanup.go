package workers

import (
	"context"
	"path"
	"time"

	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FileBatchSize   = 100
	FolderBatchSize = 50
)

type TrashCleanupWorker struct {
	DB                 *gorm.DB
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

	select {
	case <-ctx.Done():
		return 0, nil
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

	payloads := make([]events.TrashExpirationPayload, 0, len(files))
	for _, file := range files {
		markerPath := path.Join("trash", file.BucketID.String(), "files", file.ID.String())
		payloads = append(payloads, events.NewTrashExpirationFromBucketEvent(file.BucketID, markerPath).Payload)
	}

	if err := outbox.AddEvents(w.DB, payloads...); err != nil {
		return 0, err
	}

	return len(payloads), nil
}

func (w *TrashCleanupWorker) cleanupExpiredFolders(_ context.Context) (int, error) {
	expirationTime := time.Now().AddDate(0, 0, -w.TrashRetentionDays)

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

	if err := w.queueFolderPurges(folders); err != nil {
		return 0, err
	}

	return len(folders), w.cleanupOrphanedFolders(expirationTime)
}

func (w *TrashCleanupWorker) cleanupOrphanedFolders(expirationTime time.Time) error {
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

	return w.queueFolderPurges(folders)
}

func (w *TrashCleanupWorker) queueFolderPurges(folders []models.Folder) error {
	payloads := make([]events.FolderPurgePayload, 0, len(folders))
	for _, folder := range folders {
		payloads = append(payloads, events.NewFolderPurge(folder.BucketID, folder.ID, uuid.Nil).Payload)
	}
	return outbox.AddEvents(w.DB, payloads...)
}
