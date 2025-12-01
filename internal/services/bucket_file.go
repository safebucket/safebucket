package services

import (
	"errors"
	"path"
	"path/filepath"

	"api/internal/activity"
	c "api/internal/configuration"
	apierrors "api/internal/errors"
	"api/internal/handlers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/sql"
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
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
			Patch("/", handlers.UpdateHandler(s.PatchFile))

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

	// Check if folder exists (if folder_id is provided)
	if body.FolderID != nil {
		var folder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ?", body.FolderID, bucket.ID).Find(&folder)
		if result.RowsAffected == 0 {
			return models.FileTransferResponse{}, apierrors.NewAPIError(404, "FOLDER_NOT_FOUND")
		}
	}

	// Check for duplicate file name in the same folder
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

	switch body.Status {
	case models.FileStatusTrashed:
		return s.TrashFile(logger, user, bucketID, fileID)
	case models.FileStatusUploaded:
		return s.RestoreFile(logger, user, bucketID, fileID)
	case models.FileStatusUploading, models.FileStatusDeleting, models.FileStatusRestoring, models.FileStatusDeleted:
		return apierrors.NewAPIError(400, "INVALID_STATUS")
	default:
		return apierrors.NewAPIError(400, "INVALID_STATUS")
	}
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

	if file.Status == models.FileStatusTrashed {
		return models.FileTransferResponse{}, apierrors.NewAPIError(
			403,
			apierrors.ErrCannotDownloadTrashed,
		)
	}

	// Use new path structure: buckets/{bucket_id}/{file_id}
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
	bucketID uuid.UUID,
	fileID uuid.UUID,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", fileID, bucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			logger.Error("Failed to fetch file for trashing", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "FETCH_FAILED")
		}

		// Check current status - only allow uploaded → trashed transition
		if file.Status == models.FileStatusTrashed {
			return apierrors.NewAPIError(409, "FILE_ALREADY_TRASHED")
		}
		if file.Status != models.FileStatusUploaded {
			return apierrors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
		}

		// Update status to trashed and set deleted_by for audit trail
		updates := map[string]interface{}{
			"status":     models.FileStatusTrashed,
			"deleted_by": user.UserID,
		}
		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to update file status to trashed", zap.Error(err))
			return apierrors.NewAPIError(500, "UPDATE_FAILED")
		}

		// Soft delete using GORM (sets deleted_at)
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

// RestoreFile recovers a file from trash with atomic status transition.
func (s BucketFileService) RestoreFile(
	logger *zap.Logger,
	user models.UserClaims,
	bucketID, fileID uuid.UUID,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var file models.File
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ? AND status = 'trashed' ", fileID, bucketID).
			First(&file)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return apierrors.NewAPIError(404, "FILE_NOT_FOUND")
			}
			logger.Error("Failed to fetch file for restoring", zap.Error(result.Error))
			return apierrors.NewAPIError(500, "FETCH_FAILED")
		}

		// Check current status - only allow trashed → uploaded transition
		if file.Status == models.FileStatusUploaded {
			return apierrors.NewAPIError(409, "FILE_ALREADY_RESTORED")
		}
		if file.Status == models.FileStatusRestoring {
			return apierrors.NewAPIError(409, "FILE_RESTORE_IN_PROGRESS")
		}
		if file.Status != models.FileStatusTrashed {
			return apierrors.NewAPIError(409, "INVALID_FILE_STATUS_TRANSITION")
		}

		// Check for naming conflicts in the same folder
		var existingFile models.File
		query := tx.Where(
			"bucket_id = ? AND name = ? AND status != ? AND id != ?",
			file.BucketID, file.Name, models.FileStatusTrashed, file.ID,
		)
		if file.FolderID != nil {
			query = query.Where("folder_id = ?", file.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		conflictResult := query.First(&existingFile)

		if conflictResult.RowsAffected > 0 {
			return apierrors.NewAPIError(409, "FILE_NAME_CONFLICT")
		}

		// Clear soft delete and restore status (use Unscoped to update soft-deleted record)
		updates := map[string]interface{}{
			"status":     models.FileStatusUploaded,
			"deleted_at": nil,
			"deleted_by": nil,
		}

		if err := tx.Unscoped().Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to restore file status", zap.Error(err))
			return apierrors.NewAPIError(500, "UPDATE_FAILED")
		}

		// Use the new path structure: buckets/{bucket_id}/{file_id}
		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())
		if err := s.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			logger.Error(
				"Failed to unmark file as trashed - rolling back transaction",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", file.ID.String()),
			)
			return err
		}

		action := models.Activity{
			Message: activity.FileRestored,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionRestore.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"domain":      c.DefaultDomain,
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}
		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log restore activity", zap.Error(err))
			return err
		}
		return nil
	})
}

// PurgeFile permanently deletes a file from trash with atomic status transition.
func (s BucketFileService) PurgeFile(
	logger *zap.Logger,
	user models.UserClaims,
	bucketID, fileID uuid.UUID,
) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// Fetch file inside transaction with row lock (use Unscoped to query soft-deleted files)
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

		// Only allow purging trashed files
		if file.Status != models.FileStatusTrashed {
			return apierrors.NewAPIError(409, "FILE_NOT_IN_TRASH")
		}

		// Use new path structure: buckets/{bucket_id}/{file_id}
		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		// Delete the trash marker first
		if err := s.Storage.UnmarkAsTrashed(objectPath, file); err != nil {
			logger.Warn("Failed to delete trash marker",
				zap.Error(err),
				zap.String("path", objectPath))
			// Continue - marker might have been already deleted by lifecycle policy
		}

		// Delete the original file from storage
		if err := s.Storage.RemoveObject(objectPath); err != nil {
			logger.Warn("Failed to delete file from storage",
				zap.Error(err),
				zap.String("path", objectPath))
			// Continue to database deletion even if storage fails
		}

		// Hard delete from database (permanent removal)
		if err := tx.Unscoped().Delete(&file).Error; err != nil {
			logger.Error("Failed to hard delete file from database", zap.Error(err))
			return apierrors.ErrDeleteFailed
		}

		// Log activity
		action := models.Activity{
			Message: activity.FilePurged,
			Object:  file.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionPurge.String(),
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
