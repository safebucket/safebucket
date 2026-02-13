package services

import (
	"errors"
	"path"
	"path/filepath"
	"time"

	"api/internal/activity"
	apierrors "api/internal/errors"
	"api/internal/handlers"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/sql"
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BucketFileService struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	ActivityLogger     activity.IActivityLogger
	TrashRetentionDays int
}

func (s BucketFileService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FileTransferBody]).
		Post("/files", handlers.CreateHandler(s.UploadFile))

	r.Route("/files/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FilePatchBody]).
			Patch("/", handlers.BodyHandler(s.PatchFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/download", handlers.GetOneHandler(s.DownloadFile))
	})

	return r
}

func (s BucketFileService) UploadFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FileTransferBody,
) (models.FileTransferResponse, error) {
	var bucket models.Bucket
	result := s.DB.Where("id = ?", ids[0]).Find(&bucket)
	if result.RowsAffected == 0 {
		return models.FileTransferResponse{}, apierrors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	if body.FolderID != nil {
		var folder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ?", body.FolderID, bucket.ID).Find(&folder)
		if result.RowsAffected == 0 {
			return models.FileTransferResponse{}, apierrors.NewAPIError(404, "FOLDER_NOT_FOUND")
		}
	}

	var existingFile models.File
	query := s.DB.Where("bucket_id = ? AND name = ?", bucket.ID, body.Name)
	if body.FolderID != nil {
		query = query.Where("folder_id = ?", body.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFile)
	if result.RowsAffected > 0 {
		return models.FileTransferResponse{}, apierrors.NewAPIError(409, "FILE_ALREADY_EXISTS")
	}

	extension := filepath.Ext(body.Name)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	file := &models.File{
		Status:    models.FileStatusUploading,
		Name:      body.Name,
		Extension: extension,
		BucketID:  bucket.ID,
		FolderID:  body.FolderID,
		Size:      body.Size,
		ExpiresAt: body.ExpiresAt,
	}

	var url string
	var formData map[string]string
	var err error
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Create(file)
		if res.Error != nil {
			return res.Error
		}

		url, formData, err = s.Storage.PresignedPostPolicy(
			path.Join("buckets", bucket.ID.String(), file.ID.String()),
			body.Size,
			map[string]string{
				"bucket_id": bucket.ID.String(),
				"file_id":   file.ID.String(),
				"user_id":   user.UserID.String(),
			},
		)
		if err != nil {
			logger.Error("Generate presigned URL failed", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return models.FileTransferResponse{}, apierrors.ErrCreateFailed
	}

	return models.FileTransferResponse{
		ID:   file.ID.String(),
		URL:  url,
		Body: formData,
	}, nil
}

func (s BucketFileService) PatchFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FilePatchBody,
) error {
	bucketID, fileID := ids[0], ids[1]

	var file models.File
	result := s.DB.Unscoped().
		Where("id = ? AND bucket_id = ?", fileID, bucketID).
		Find(&file)
	if result.Error != nil {
		return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}
	if result.RowsAffected == 0 {
		return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
	}

	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return apierrors.NewAPIError(403, apierrors.ErrFileExpired)
	}

	switch body.Status {
	case string(models.FileStatusDeleted):
		return s.TrashFile(logger, user, file)
	case string(models.FileStatusUploaded):
		if file.DeletedAt.Valid {
			return s.RestoreFile(logger, user, file)
		}
		return s.HandleUploadedStatus(logger, user, file)
	default:
		return apierrors.NewAPIError(400, "INVALID_STATUS")
	}
}

// HandleUploadedStatus confirms a file upload by transitioning from "uploading" to "uploaded".
// This is required for S3 providers that don't support bucket notifications.
// The client must call this after completing the upload via the presigned URL.
func (s BucketFileService) HandleUploadedStatus(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		if file.Status != models.FileStatusUploading {
			return apierrors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())
		if _, err := s.Storage.StatObject(objectPath); err != nil {
			logger.Error("File not found in storage",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()))
			return apierrors.NewAPIError(404, "FILE_NOT_IN_STORAGE")
		}

		if err := tx.Model(&file).Update("status", models.FileStatusUploaded).Error; err != nil {
			logger.Error("Failed to update file status", zap.Error(err))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		if err := s.ActivityLogger.Send(models.Activity{
			Message: activity.FileUploaded,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionCreate.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}); err != nil {
			logger.Warn("Failed to log upload activity", zap.Error(err))
		}

		return nil
	})
}

// DeleteFile handles DELETE requests for permanent file deletion (purge).
func (s BucketFileService) DeleteFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	return s.PurgeFile(logger, user, bucketID, fileID)
}

func (s BucketFileService) DownloadFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) (models.FileTransferResponse, error) {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	if file.DeletedAt.Valid {
		return models.FileTransferResponse{}, apierrors.NewAPIError(
			403,
			apierrors.ErrCannotDownloadTrashed,
		)
	}

	if file.ExpiresAt != nil && file.ExpiresAt.Before(time.Now()) {
		return models.FileTransferResponse{}, apierrors.NewAPIError(
			403,
			apierrors.ErrFileExpired,
		)
	}

	url, err := s.Storage.PresignedGetObject(
		path.Join("buckets", file.BucketID.String(), file.ID.String()),
	)
	if err != nil {
		logger.Error("Generate presigned URL failed", zap.Error(err))
		return models.FileTransferResponse{}, err
	}

	action := models.Activity{
		Message: activity.FileDownloaded,
		Object:  file.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDownload.String(),
			"bucket_id":   bucketID.String(),
			"file_id":     fileID.String(),
			"object_type": rbac.ResourceFile.String(),
			"user_id":     user.UserID.String(),
		}),
	}
	err = s.ActivityLogger.Send(action)
	if err != nil {
		return models.FileTransferResponse{}, err
	}

	return models.FileTransferResponse{
		ID:  file.ID.String(),
		URL: url,
	}, nil
}

// TrashFile moves a file to trash (soft delete) with atomic status transition.
func (s BucketFileService) TrashFile(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			logger.Error("Failed to fetch file for trashing", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "FETCH_FAILED")
		}

		if file.DeletedAt.Valid {
			return apierrors.NewAPIError(409, "FILE_ALREADY_TRASHED")
		}

		if file.Status != models.FileStatusUploaded {
			return apierrors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
		}

		updates := map[string]interface{}{
			"status":     models.FileStatusDeleted,
			"deleted_by": user.UserID,
		}
		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to update file for trashing", zap.Error(err))
			return apierrors.NewAPIError(500, "UPDATE_FAILED")
		}

		if err := tx.Delete(&file).Error; err != nil {
			logger.Error("Failed to soft delete file", zap.Error(err))
			return apierrors.NewAPIError(500, "DELETE_FAILED")
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		if err := s.Storage.MarkAsTrashed(objectPath, file); err != nil {
			logger.Error(
				"Failed to mark file as trashed - rolling back transaction",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()),
			)
			return err
		}

		action := models.Activity{
			Message: activity.FileTrashed,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionErase.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log trash activity", zap.Error(err))
			return err
		}

		return nil
	})
}

func (s BucketFileService) restoreParentFolders(
	tx *gorm.DB,
	logger *zap.Logger,
	folderID *uuid.UUID,
	bucketID uuid.UUID,
) ([]models.Folder, error) {
	return h.RestoreParentFolders(tx, logger, folderID, bucketID)
}

// unmarkRestoredFolders removes trash markers for restored folders.
// This must be called AFTER the transaction commits to avoid race conditions.
func (s BucketFileService) unmarkRestoredFolders(logger *zap.Logger, folders []models.Folder) {
	for _, folder := range folders {
		objectPath := path.Join("buckets", folder.BucketID.String(), folder.ID.String())
		if err := s.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
			logger.Warn("Failed to unmark parent folder as trashed",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
			// Continue - folders exist only in DB
		}
	}
}

// RestoreFile recovers a file from trash with atomic status transition.
func (s BucketFileService) RestoreFile(
	logger *zap.Logger,
	user models.UserClaims,
	file models.File,
) error {
	var restoredFolders []models.Folder
	var restoredFile models.File

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ? AND deleted_at IS NOT NULL", file.ID, file.BucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			logger.Error("Failed to fetch file for restoring", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "FETCH_FAILED")
		}

		if file.Status == models.FileStatusRestoring {
			return apierrors.NewAPIError(409, "FILE_RESTORE_IN_PROGRESS")
		}

		folders, err := s.restoreParentFolders(tx, logger, file.FolderID, file.BucketID)
		if err != nil {
			return err
		}
		restoredFolders = folders

		var existingFile models.File
		query := tx.Where(
			"bucket_id = ? AND name = ? AND id != ?",
			file.BucketID, file.Name, file.ID,
		)
		if file.FolderID != nil {
			query = query.Where("folder_id = ?", file.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		conflictResult := query.Find(&existingFile)

		if conflictResult.RowsAffected > 0 {
			return apierrors.NewAPIError(409, "FILE_NAME_CONFLICT")
		}

		updates := map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
			"status":     models.FileStatusUploaded,
		}

		if updateErr := tx.Unscoped().Model(&file).Updates(updates).Error; updateErr != nil {
			logger.Error("Failed to restore file", zap.Error(updateErr))
			return apierrors.NewAPIError(500, "UPDATE_FAILED")
		}

		restoredFile = file

		action := models.Activity{
			Message: activity.FileRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionRestore.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}
		if activityErr := s.ActivityLogger.Send(action); activityErr != nil {
			logger.Error("Failed to log restore activity", zap.Error(activityErr))
			return activityErr
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.unmarkRestoredFolders(logger, restoredFolders)

	objectPath := path.Join("buckets", restoredFile.BucketID.String(), restoredFile.ID.String())
	if storageErr := s.Storage.UnmarkAsTrashed(objectPath, restoredFile); storageErr != nil {
		logger.Warn(
			"Failed to unmark file as trashed (file already restored in DB)",
			zap.Error(storageErr),
			zap.String("path", objectPath),
			zap.String("file_id", restoredFile.ID.String()),
		)
		// Don't return error - the database is already updated
	}

	return nil
}

// PurgeFile permanently deletes a file from trash with atomic status transition.
func (s BucketFileService) PurgeFile(
	logger *zap.Logger,
	user models.UserClaims,
	bucketID, fileID uuid.UUID,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", fileID, bucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			logger.Error("Failed to fetch file for purging", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "FETCH_FAILED")
		}

		if !file.DeletedAt.Valid {
			return apierrors.NewAPIError(409, "FILE_NOT_IN_TRASH")
		}

		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		if err := s.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			logger.Warn("Failed to delete trash marker",
				zap.Error(err),
				zap.String("path", objectPath))
		}

		if err := s.Storage.RemoveObject(objectPath); err != nil {
			logger.Warn("Failed to delete file from storage",
				zap.Error(err),
				zap.String("path", objectPath))
		}

		if err := tx.Unscoped().Delete(&file).Error; err != nil {
			logger.Error("Failed to hard delete file from database", zap.Error(err))
			return apierrors.ErrDeleteFailed
		}

		action := models.Activity{
			Message: activity.FileDeleted,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionDelete.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log purge activity", zap.Error(err))
			return err
		}

		return nil
	})
}
