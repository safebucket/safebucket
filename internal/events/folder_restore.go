package events

import (
	"errors"
	"net/http"
	"path"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	c "github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	FolderRestoreName        = "FolderRestore"
	FolderRestorePayloadName = "FolderRestorePayload"
)

type FolderRestorePayload struct {
	Type       string
	BucketID   uuid.UUID
	FolderID   uuid.UUID
	UserID     uuid.UUID
	RestoredAt time.Time
}

func (p FolderRestorePayload) EventName() string { return p.Type }

type FolderRestore struct {
	Payload FolderRestorePayload
}

func NewFolderRestore(
	bucketID uuid.UUID,
	folderID uuid.UUID,
	userID uuid.UUID,
	restoredAt time.Time,
) FolderRestore {
	return FolderRestore{
		Payload: FolderRestorePayload{
			Type:       FolderRestoreName,
			BucketID:   bucketID,
			FolderID:   folderID,
			UserID:     userID,
			RestoredAt: restoredAt,
		},
	}
}

func (e *FolderRestore) callback(params *EventParams) error {
	var folders []models.Folder
	var files []models.File

	err := params.DB.Transaction(func(tx *gorm.DB) error {
		live, err := e.lockFolder(tx)
		if err != nil || !live {
			return err
		}

		folders, err = sql.RestoreChildFolders(
			tx,
			e.Payload.BucketID,
			e.Payload.FolderID,
			e.Payload.RestoredAt,
			c.BulkActionsLimit,
		)
		if err != nil {
			return err
		}

		remaining := c.BulkActionsLimit - len(folders)
		if remaining > 0 {
			files, err = sql.RestoreChildFiles(
				tx, e.Payload.BucketID,
				e.Payload.FolderID,
				e.Payload.RestoredAt,
				remaining,
			)
			if err != nil {
				return err
			}
		}

		if err = outbox.AddActivities(tx, e.restoredActivities(folders, files)...); err != nil {
			return err
		}

		payloads := make([]FolderRestorePayload, 0, len(folders)+1)
		for _, folder := range folders {
			payloads = append(payloads,
				NewFolderRestore(e.Payload.BucketID, folder.ID, e.Payload.UserID, e.Payload.RestoredAt).Payload)
		}

		if len(folders)+len(files) > 0 {
			payloads = append(payloads, e.Payload)
		}
		return outbox.AddEvents(tx, payloads...)
	})
	if err != nil {
		zap.L().Error("Failed to restore folder content",
			zap.String("folder_id", e.Payload.FolderID.String()), zap.Error(err))
		return err
	}

	e.unmarkTrashed(params, folders, files)
	return nil
}

func (e *FolderRestore) lockFolder(tx *gorm.DB) (bool, error) {
	if err := sql.LockBucketFileNames(tx, e.Payload.BucketID); err != nil {
		apiErr, ok := errors.AsType[*apierrors.APIError](err)

		if ok && apiErr.Status == http.StatusNotFound {
			zap.L().Info("Bucket deleted, skipping folder restore",
				zap.String("bucket_id", e.Payload.BucketID.String()))
			return false, nil
		}
		return false, err
	}

	result := tx.Where("id = ? AND bucket_id = ? AND status = ?",
		e.Payload.FolderID, e.Payload.BucketID, models.FolderStatusCreated).Find(&models.Folder{})

	if result.Error != nil {
		return false, result.Error
	}

	if result.RowsAffected == 0 {
		zap.L().Info("Folder doesn't exist, stopping restore",
			zap.String("folder_id", e.Payload.FolderID.String()))
		return false, nil
	}
	return true, nil
}

func (e *FolderRestore) restoredActivities(folders []models.Folder, files []models.File) []models.Activity {
	activities := make([]models.Activity, 0, len(folders)+len(files))
	for _, folder := range folders {
		activities = append(activities, models.Activity{
			Message: activity.FolderRestored,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   folder.BucketID.String(),
				FolderID:   folder.ID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		})
	}

	for _, file := range files {
		activities = append(activities, models.Activity{
			Message: activity.FileRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionRestore.String(),
				BucketID:   file.BucketID.String(),
				FileID:     file.ID.String(),
				ObjectType: rbac.ResourceFile.String(),
				UserID:     e.Payload.UserID.String(),
			}),
		})
	}

	return activities
}

func (e *FolderRestore) unmarkTrashed(params *EventParams, folders []models.Folder, files []models.File) {
	for _, folder := range folders {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), folder.ID.String())
		if err := params.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			zap.L().Warn("Failed to unmark folder as trashed",
				zap.Error(err), zap.String("folder_id", folder.ID.String()))
		}
	}

	for _, file := range files {
		objectPath := path.Join("buckets", e.Payload.BucketID.String(), file.ID.String())
		if err := params.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			zap.L().Warn("Failed to unmark file as trashed",
				zap.Error(err), zap.String("file_id", file.ID.String()))
		}
	}
}
