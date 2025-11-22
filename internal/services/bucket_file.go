package services

import (
	"path"
	"path/filepath"
	"time"

	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/errors"
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

	// File upload
	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FileTransferBody]).
		Post("/files", handlers.CreateHandler(s.UploadFile))

	// Trash management
	r.Route("/trash", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/", handlers.GetOneHandler(s.ListTrashedItems))
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/{id1}", handlers.DeleteHandler(s.PurgeFile))
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Post("/{id1}/restore", handlers.DeleteHandler(s.RestoreFile))
	})

	// File operations
	r.Route("/files/{id1}", func(r chi.Router) {
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
		return models.FileTransferResponse{}, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	// Check if folder exists (if folder_id is provided)
	if body.FolderID != nil {
		var folder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ?", body.FolderID, bucket.ID).Find(&folder)
		if result.RowsAffected == 0 {
			return models.FileTransferResponse{}, errors.NewAPIError(404, "FOLDER_NOT_FOUND")
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
		return models.FileTransferResponse{}, errors.NewAPIError(409, "FILE_ALREADY_EXISTS")
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
		return models.FileTransferResponse{}, errors.ErrCreateFailed
	}

	return models.FileTransferResponse{
		ID:   file.ID.String(),
		URL:  url,
		Body: formData,
	}, nil
}

func (s BucketFileService) DeleteFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	return s.TrashFile(logger, user, file)
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
		return models.FileTransferResponse{}, errors.NewAPIError(
			403,
			errors.ErrCannotDownloadTrashed,
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
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionDownload.String(),
			"bucket_id":   bucketID.String(),
			"file_id":     fileID.String(),
			"domain":      c.DefaultDomain,
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

// TrashFile moves a file to trash (soft delete).
func (s BucketFileService) TrashFile(logger *zap.Logger, user models.UserClaims, file models.File) error {
	if file.Status != models.FileStatusUploaded {
		return errors.NewAPIError(400, errors.ErrFileCannotBeTrashed)
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		updates := map[string]interface{}{
			"status":     models.FileStatusTrashed,
			"trashed_at": now,
			"trashed_by": user.UserID,
		}

		if err := tx.Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to update file status to trashed", zap.Error(err))
			return errors.NewAPIError(500, "UPDATE_FAILED")
		}

		// Use new path structure: buckets/{bucket_id}/{file_id}
		objectPath := path.Join("buckets", file.BucketID.String(), file.ID.String())

		if err := s.Storage.MarkFileAsTrashed(objectPath, models.TrashMetadata{
			TrashedAt: now,
			TrashedBy: user.UserID,
			ObjectID:  file.ID,
			IsFolder:  false,
		}); err != nil {
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
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionErase.String(),
				"bucket_id":   file.BucketID.String(),
				"file_id":     file.ID.String(),
				"domain":      c.DefaultDomain,
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

// RestoreFile recovers a file from trash.
func (s BucketFileService) RestoreFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	if file.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFileNotInTrash)
	}
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	if file.TrashedAt != nil && time.Since(*file.TrashedAt) > retentionPeriod {
		return errors.NewAPIError(410, errors.ErrFileTrashExpired)
	}

	// Check for naming conflicts in the same folder
	var existingFile models.File
	query := s.DB.Where(
		"bucket_id = ? AND name = ? AND status != ?",
		bucketID, file.Name, models.FileStatusTrashed,
	)
	if file.FolderID != nil {
		query = query.Where("folder_id = ?", file.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result := query.First(&existingFile)

	if result.RowsAffected > 0 {
		return errors.NewAPIError(409, "FILE_NAME_CONFLICT")
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":     models.FileStatusUploaded,
			"trashed_at": nil,
			"trashed_by": nil,
		}

		if err = tx.Model(&file).Updates(updates).Error; err != nil {
			logger.Error("Failed to restore file status", zap.Error(err))
			return errors.NewAPIError(500, "UPDATE_FAILED")
		}

		// Use new path structure: buckets/{bucket_id}/{file_id}
		objectPath := path.Join("buckets", bucketID.String(), fileID.String())
		if err = s.Storage.UnmarkFileAsTrashed(objectPath); err != nil {
			logger.Error(
				"Failed to unmark file as trashed - rolling back transaction",
				zap.Error(err),
				zap.String("path", objectPath),
				zap.String("file_id", fileID.String()),
			)
			return err
		}

		action := models.Activity{
			Message: activity.FileRestored,
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionRestore.String(),
				"bucket_id":   bucketID.String(),
				"file_id":     fileID.String(),
				"domain":      c.DefaultDomain,
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}
		if err = s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log restore activity", zap.Error(err))
			return err
		}
		return nil
	})
}

// TrashResponse holds both trashed files and folders.
type TrashResponse struct {
	Files   []models.File   `json:"files"`
	Folders []models.Folder `json:"folders"`
}

// ListTrashedItems returns all trashed files and folders for a bucket within retention window.
func (s BucketFileService) ListTrashedItems(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (TrashResponse, error) {
	bucketID := ids[0]
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	cutoffDate := time.Now().Add(-retentionPeriod)

	var files []models.File
	var folders []models.Folder

	// Fetch trashed files
	fileResult := s.DB.
		Preload("TrashedUser").
		Where(
			"bucket_id = ? AND status = ? AND trashed_at > ?",
			bucketID,
			models.FileStatusTrashed,
			cutoffDate,
		).
		Order("trashed_at DESC").
		Find(&files)

	if fileResult.Error != nil {
		logger.Error("Failed to list trashed files", zap.Error(fileResult.Error))
		files = []models.File{}
	}

	// Fetch trashed folders
	folderResult := s.DB.
		Preload("TrashedUser").
		Where(
			"bucket_id = ? AND status = ? AND trashed_at > ?",
			bucketID,
			models.FileStatusTrashed,
			cutoffDate,
		).
		Order("trashed_at DESC").
		Find(&folders)

	if folderResult.Error != nil {
		logger.Error("Failed to list trashed folders", zap.Error(folderResult.Error))
		folders = []models.Folder{}
	}

	return TrashResponse{
		Files:   files,
		Folders: folders,
	}, nil
}

// PurgeFile permanently deletes a file from trash (hard delete).
func (s BucketFileService) PurgeFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	// Validate file is in trash
	if file.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFileNotInTrash)
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		// Use new path structure: buckets/{bucket_id}/{file_id}
		objectPath := path.Join("buckets", bucketID.String(), fileID.String())

		// Delete the trash marker first
		if err = s.Storage.UnmarkFileAsTrashed(objectPath); err != nil {
			logger.Warn("Failed to delete trash marker",
				zap.Error(err),
				zap.String("path", objectPath))
			// Continue - marker might have been already deleted by lifecycle policy
		}

		// Delete the original file from storage
		if err = s.Storage.RemoveObject(objectPath); err != nil {
			logger.Warn("Failed to delete file from storage",
				zap.Error(err),
				zap.String("path", objectPath))
			// Continue to database deletion even if storage fails
		}

		// Soft delete from database (allows activity enrichment to still find the record)
		if err = tx.Delete(&file).Error; err != nil {
			logger.Error("Failed to soft delete file from database", zap.Error(err))
			return errors.ErrDeleteFailed
		}

		// Log activity
		action := models.Activity{
			Message: activity.FilePurged,
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionPurge.String(),
				"bucket_id":   bucketID.String(),
				"file_id":     fileID.String(),
				"domain":      c.DefaultDomain,
				"object_type": rbac.ResourceFile.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		if err = s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log purge activity", zap.Error(err))
			return err
		}

		return nil
	})
}
