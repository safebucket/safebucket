package workers

import (
	"context"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/fileversions"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"gorm.io/gorm"
)

type TrashCleanupWorker struct {
	DB                 *gorm.DB
	Versions           fileversions.Manager
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
	expiredBefore := time.Now().AddDate(0, 0, -w.TrashRetentionDays)
	total := 0

	for ctx.Err() == nil {
		var files []models.File
		err := w.DB.Transaction(func(tx *gorm.DB) error {
			var err error
			files, err = sql.MarkExpiredFilesDeleting(tx, expiredBefore, c.BatchLimit)
			if err != nil || len(files) == 0 {
				return err
			}
			return outbox.AddActivities(tx, fileDeletedActivities(files)...)
		})
		if err != nil || len(files) == 0 {
			return total, err
		}

		if err = w.Versions.DeleteFiles(files...); err != nil {
			return total, err
		}

		total += len(files)
	}
	return total, nil
}

func (w *TrashCleanupWorker) cleanupExpiredFolders(ctx context.Context) (int, error) {
	expiredBefore := time.Now().AddDate(0, 0, -w.TrashRetentionDays)
	total := 0

	for ctx.Err() == nil {
		var folders []models.Folder
		err := w.DB.Transaction(func(tx *gorm.DB) error {
			var err error
			folders, err = sql.DeleteExpiredEmptyFolders(tx, expiredBefore, c.BatchLimit)
			if err != nil || len(folders) == 0 {
				return err
			}
			return outbox.AddActivities(tx, folderDeletedActivities(folders)...)
		})
		if err != nil || len(folders) == 0 {
			return total, err
		}

		total += len(folders)
	}
	return total, nil
}

func fileDeletedActivities(files []models.File) []models.Activity {
	activities := make([]models.Activity, 0, len(files))
	for _, file := range files {
		activities = append(activities, models.Activity{
			Message: activity.FileDeleted,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
			}),
		})
	}
	return activities
}

func folderDeletedActivities(folders []models.Folder) []models.Activity {
	activities := make([]models.Activity, 0, len(folders))
	for _, folder := range folders {
		activities = append(activities, models.Activity{
			Message: activity.FolderDeleted,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionDelete.String(),
				BucketID:   folder.BucketID.String(),
				FolderID:   folder.ID.String(),
				ObjectType: rbac.ResourceFolder.String(),
			}),
		})
	}
	return activities
}
