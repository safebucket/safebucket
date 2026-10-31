package services

import (
	"errors"
	"net/http"
	"path"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BucketFolderService struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	TrashRetentionDays int
}

func (s BucketFolderService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FolderCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateFolder))

	r.Route("/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FolderUpdateBody]).
			Put("/", handlers.BodyHandler(s.RenameFolder))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FolderPatchBody]).
			Patch("/", handlers.BodyHandler(s.UpdateFolderStatus))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteFolder))
	})

	return r
}

func (s BucketFolderService) CreateFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FolderCreateBody,
) (models.Folder, error) {
	bucketID := ids[0]

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketID).Find(&bucket)
	if result.RowsAffected == 0 {
		return models.Folder{}, apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
	}

	if body.FolderID != nil {
		var parentFolder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ? AND status = ?",
			body.FolderID, bucketID, models.FolderStatusCreated).Find(&parentFolder)
		if result.RowsAffected == 0 {
			return models.Folder{}, apierrors.New(http.StatusNotFound, apierrors.CodeParentFolderNotFound)
		}
	}

	var existingFolder models.Folder
	query := s.DB.Where("bucket_id = ? AND name = ?", bucketID, body.Name)
	if body.FolderID != nil {
		query = query.Where("folder_id = ?", body.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFolder)
	if result.RowsAffected > 0 {
		return models.Folder{}, apierrors.New(http.StatusConflict, apierrors.CodeFolderAlreadyExists)
	}

	folder := models.Folder{
		Name:     body.Name,
		Status:   models.FolderStatusCreated,
		BucketID: bucketID,
		FolderID: body.FolderID,
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&folder).Error; createErr != nil {
			logger.Error("Failed to create folder", zap.Error(createErr))
			return createErr
		}

		action := models.Activity{
			Message: activity.FolderCreated,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionCreate.String(),
				BucketID:   bucketID.String(),
				FolderID:   folder.ID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     user.UserID.String(),
			}),
		}

		if activityErr := outbox.AddActivities(tx, action); activityErr != nil {
			logger.Error("Failed to log folder creation activity", zap.Error(activityErr))
			return activityErr
		}

		return nil
	})
	if err != nil {
		return models.Folder{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeCreateFailed)
	}

	return folder, nil
}

func (s BucketFolderService) RenameFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FolderUpdateBody,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Where("id = ? AND bucket_id = ? AND status = ?",
		folderID, bucketID, models.FolderStatusCreated).First(&folder)
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
	}

	var existingFolder models.Folder
	query := s.DB.Where("bucket_id = ? AND name = ? AND id != ?", bucketID, body.Name, folderID)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFolder)
	if result.RowsAffected > 0 {
		return apierrors.New(http.StatusConflict, apierrors.CodeFolderNameConflict)
	}

	folder.Name = body.Name

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if updateErr := tx.Model(&folder).Update("name", body.Name).Error; updateErr != nil {
			logger.Error("Failed to update folder", zap.Error(updateErr))
			return updateErr
		}

		action := models.Activity{
			Message: activity.FolderUpdated,
			Object:  folder.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     rbac.ActionUpdate.String(),
				BucketID:   bucketID.String(),
				FolderID:   folderID.String(),
				ObjectType: rbac.ResourceFolder.String(),
				UserID:     user.UserID.String(),
			}),
		}

		if activityErr := outbox.AddActivities(tx, action); activityErr != nil {
			logger.Error("Failed to log folder update activity", zap.Error(activityErr))
			return activityErr
		}

		return nil
	})
	if err != nil {
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
	}

	return nil
}

func (s BucketFolderService) UpdateFolderStatus(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FolderPatchBody,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Unscoped().Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
	}

	if body.Status != models.FolderStatusCreated {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeInvalidStatus)
	}

	return s.RestoreFolder(logger, user, folder)
}

func (s BucketFolderService) DeleteFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", folderID, bucketID).
			First(&folder)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
			}
			logger.Error("Failed to fetch folder for purging", zap.Error(result.Error))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeFetchFailed)
		}

		if !folder.DeletedAt.Valid {
			return apierrors.New(http.StatusConflict, apierrors.CodeFolderNotInTrash)
		}

		if folder.Status == models.FolderStatusRestoring {
			return apierrors.New(http.StatusConflict, apierrors.CodeFolderRestoreInProgress)
		}

		if folder.Status != models.FolderStatusDeleted {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidStatus)
		}

		event := events.NewFolderPurge(folder.BucketID, folder.ID, user.UserID)
		if eventErr := outbox.AddEvents(tx, events.FolderPurgeName, event.Payload); eventErr != nil {
			logger.Error("Failed to queue folder purge", zap.Error(eventErr))
			return eventErr
		}

		return nil
	})

	if err != nil {
		return err
	}

	logger.Info("Folder purge initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

func (s BucketFolderService) restoreParentFolders(
	tx *gorm.DB,
	logger *zap.Logger,
	folderID *uuid.UUID,
	bucketID uuid.UUID,
) ([]models.Folder, error) {
	return h.RestoreParentFolders(tx, logger, folderID, bucketID)
}

func (s BucketFolderService) unmarkRestoredFolders(logger *zap.Logger, folders []models.Folder) {
	for _, folder := range folders {
		objectPath := path.Join("buckets", folder.BucketID.String(), folder.ID.String())
		if err := s.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			logger.Warn("Failed to unmark parent folder as trashed",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
		}
	}
}

func (s BucketFolderService) RestoreFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	var restoredParentFolders []models.Folder
	var restoredFolder models.Folder

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var lockedFolder models.Folder
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", folder.ID, folder.BucketID).
			First(&lockedFolder)

		if result.Error != nil {
			return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
		}

		if !lockedFolder.DeletedAt.Valid {
			return apierrors.New(http.StatusConflict, apierrors.CodeFolderNotInTrash)
		}

		if lockedFolder.Status == models.FolderStatusRestoring {
			return apierrors.New(http.StatusConflict, apierrors.CodeFolderRestoreInProgress)
		}

		retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
		if time.Since(lockedFolder.DeletedAt.Time) > retentionPeriod {
			return apierrors.New(http.StatusGone, apierrors.CodeFolderTrashExpired)
		}

		parentFolders, err := s.restoreParentFolders(tx, logger, lockedFolder.FolderID, lockedFolder.BucketID)
		if err != nil {
			return err
		}
		restoredParentFolders = parentFolders

		var existingFolder models.Folder
		query := tx.Where(
			"bucket_id = ? AND name = ? AND id != ?",
			lockedFolder.BucketID,
			lockedFolder.Name,
			lockedFolder.ID,
		)
		if lockedFolder.FolderID != nil {
			query = query.Where("folder_id = ?", lockedFolder.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		if query.Find(&existingFolder); existingFolder.ID != uuid.Nil {
			return apierrors.New(http.StatusConflict, apierrors.CodeFolderNameConflict)
		}

		if updateErr := tx.Unscoped().
			Model(&lockedFolder).
			Update("status", models.FolderStatusRestoring).
			Error; updateErr != nil {
			logger.Error("Failed to set folder to restoring status", zap.Error(updateErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeUpdateFailed)
		}

		restoredFolder = lockedFolder

		event := events.NewFolderRestore(restoredFolder.BucketID, restoredFolder.ID, user.UserID)
		if eventErr := outbox.AddEvents(tx, events.FolderRestoreName, event.Payload); eventErr != nil {
			logger.Error("Failed to queue folder restore", zap.Error(eventErr))
			return eventErr
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.unmarkRestoredFolders(logger, restoredParentFolders)

	objectPath := path.Join("buckets", restoredFolder.BucketID.String(), restoredFolder.ID.String())
	if storageErr := s.Storage.UnmarkAsTrashed(objectPath, restoredFolder); storageErr != nil {
		logger.Warn("Failed to remove trash marker for folder", zap.Error(storageErr))
	}

	logger.Info("Folder restore initiated (async)",
		zap.String("folder", restoredFolder.Name),
		zap.String("folder_id", restoredFolder.ID.String()))

	return nil
}

func (s BucketFolderService) ListTrashedFolders(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.Folder {
	var folders []models.Folder
	result := s.DB.Unscoped().
		Where(
			"bucket_id = ? AND deleted_at IS NOT NULL AND status != ?",
			ids[0],
			models.FolderStatusRestoring,
		).
		Order("deleted_at DESC").
		Find(&folders)

	if result.Error != nil {
		logger.Error("Failed to list trashed folders", zap.Error(result.Error))
		return []models.Folder{}
	}

	return folders
}
