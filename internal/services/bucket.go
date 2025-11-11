package services

import (
	"context"
	"path"
	"path/filepath"
	"time"

	"api/internal/activity"
	c "api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/messaging"
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

type BucketService struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	Publisher          messaging.IPublisher
	Providers          c.Providers
	ActivityLogger     activity.IActivityLogger
	WebURL             string
	TrashRetentionDays int
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleGuest)).
		Get("/", handlers.GetListHandler(s.GetBucketList))

	r.With(m.AuthorizeRole(models.RoleUser)).
		With(m.Validate[models.BucketCreateUpdateBody]).
		Post("/", handlers.CreateHandler(s.CreateBucket))

	r.With(m.AuthorizeRole(models.RoleGuest)).
		Get("/activity", handlers.GetListHandler(s.GetActivity))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/", handlers.GetOneHandler(s.GetBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			With(m.Validate[models.BucketCreateUpdateBody]).
			Patch("/", handlers.UpdateHandler(s.UpdateBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteBucket))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FileTransferBody]).
			Post("/files", handlers.CreateHandler(s.UploadFile))

		r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
			Get("/activity", handlers.GetOneHandler(s.GetBucketActivity))

		r.Mount("/members", BucketMemberService{
			DB:             s.DB,
			Providers:      s.Providers,
			Publisher:      s.Publisher,
			ActivityLogger: s.ActivityLogger,
			WebURL:         s.WebURL,
		}.Routes())

		r.Route("/trash", func(r chi.Router) {
			r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
				Get("/", handlers.GetListHandler(s.ListTrashedFiles))
			r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
				Delete("/{id1}", handlers.DeleteHandler(s.PurgeFile))
			r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
				Post("/{id1}/restore", handlers.DeleteHandler(s.RestoreFile))
		})

		r.Route("/files/{id1}", func(r chi.Router) {
			r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
				Delete("/", handlers.DeleteHandler(s.DeleteFile))

			r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
				Get("/download", handlers.GetOneHandler(s.DownloadFile))
		})
	})

	return r
}

func (s BucketService) CreateBucket(
	logger *zap.Logger,
	user models.UserClaims,
	_ uuid.UUIDs,
	body models.BucketCreateUpdateBody,
) (models.Bucket, error) {
	var newBucket models.Bucket

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		newBucket = models.Bucket{Name: body.Name, CreatedBy: user.UserID}
		res := tx.Create(&newBucket)

		if res.Error != nil {
			logger.Error("Failed to create bucket", zap.Error(res.Error))
			return res.Error
		}

		err := rbac.CreateMembership(tx, user.UserID, newBucket.ID, models.GroupOwner)
		if err != nil {
			logger.Error("Failed to create owner membership", zap.Error(err))
			return err
		}

		action := models.Activity{
			Message: activity.BucketCreated,
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionCreate.String(),
				"domain":      c.DefaultDomain,
				"object_type": rbac.ResourceBucket.String(),
				"bucket_id":   newBucket.ID.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		err = s.ActivityLogger.Send(action)
		if err != nil {
			logger.Error("Failed to register activity", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return models.Bucket{}, errors.ErrCreateFailed
	}

	return newBucket, nil
}

func (s BucketService) GetBucketList(
	logger *zap.Logger,
	user models.UserClaims,
	_ uuid.UUIDs,
) []models.Bucket {
	var buckets []models.Bucket
	if !user.Valid() {
		logger.Warn("Invalid user claims", zap.String("user_id", user.UserID.String()))
		return []models.Bucket{}
	}

	memberships, err := rbac.GetUserBuckets(s.DB, user.UserID)
	if err != nil {
		logger.Error(
			"Error retrieving user buckets",
			zap.Error(err),
			zap.String("user_id", user.UserID.String()),
		)
		return []models.Bucket{}
	}

	var bucketIDs []uuid.UUID
	for _, membership := range memberships {
		bucketIDs = append(bucketIDs, membership.BucketID)
	}

	if len(bucketIDs) == 0 {
		return []models.Bucket{}
	}

	if err = s.DB.Where("id IN ?", bucketIDs).Find(&buckets).Error; err != nil {
		logger.Error("Error querying buckets", zap.Error(err))
		return []models.Bucket{}
	}

	return buckets
}

func (s BucketService) GetBucket(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (models.Bucket, error) {
	bucketID := ids[0]
	var bucket models.Bucket
	bucket.Files = []models.File{}

	result := s.DB.Where("id = ?", bucketID).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	var files []models.File
	// Filter out expired files that haven't been uploaded yet (only applies to files, not folders)
	// Also exclude trashed items from normal bucket view
	expirationTime := time.Now().Add(-c.UploadPolicyExpirationInMinutes * time.Minute)
	result = s.DB.Where(
		"bucket_id = ? AND (status IS NULL OR status != ?) AND (type = 'folder' OR status = ? OR (status = ? AND created_at > ?))",
		bucketID,
		models.FileStatusTrashed,
		models.FileStatusUploaded,
		models.FileStatusUploading,
		expirationTime,
	).Find(&files)

	if result.RowsAffected > 0 {
		bucket.Files = files
	}
	return bucket, nil
}

func (s BucketService) UpdateBucket(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.BucketCreateUpdateBody,
) error {
	bucket := models.Bucket{ID: ids[0]}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}
	return nil
}

func (s BucketService) DeleteBucket(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		bucket := models.Bucket{}
		result := tx.Where("id = ?", ids[0]).First(&bucket)

		if result.RowsAffected == 0 {
			return errors.NewAPIError(404, "BUCKET_NOT_FOUND")
		}

		// Soft delete bucket (memberships will be cascade deleted by foreign key constraint)
		if _, err := gorm.G[models.Bucket](tx).Where("id = ?", bucket.ID).Delete(context.Background()); err != nil {
			return err
		}

		// Hard delete all invitations associated to the bucket
		if _, err := gorm.G[models.Invite](tx).Where("bucket_id = ?", bucket.ID).Delete(context.Background()); err != nil {
			return err
		}

		action := models.Activity{
			Message: activity.BucketDeleted,
			Filter: activity.NewLogFilter(map[string]string{
				"action":      rbac.ActionDelete.String(),
				"bucket_id":   bucket.ID.String(),
				"domain":      c.DefaultDomain,
				"object_type": rbac.ResourceBucket.String(),
				"user_id":     user.UserID.String(),
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			return err
		}

		// Trigger async file and folder deletion from root path
		event := events.NewObjectDeletion(s.Publisher, bucket, "/")
		event.Trigger()
		return nil
	})
	if err != nil {
		logger.Error("Failed to delete bucket", zap.Error(err))
		return errors.ErrDeleteFailed
	}

	return nil
}

func (s BucketService) UploadFile(
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

	var existingFile models.File
	result = s.DB.Where("bucket_id = ? AND name = ? AND path = ?", bucket.ID, body.Name, body.Path).
		Find(&existingFile)
	if result.RowsAffected > 0 {
		return models.FileTransferResponse{}, errors.NewAPIError(409, "FILE_ALREADY_EXISTS")
	}

	extension := filepath.Ext(body.Name)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	var status models.FileStatus
	if body.Type == models.FileTypeFile {
		status = models.FileStatusUploading
	}

	file := &models.File{
		Status:    status,
		Name:      body.Name,
		Extension: extension,
		BucketID:  bucket.ID,
		Path:      body.Path,
		Type:      body.Type,
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

		if file.Type == models.FileTypeFile {
			url, formData, err = s.Storage.PresignedPostPolicy(
				path.Join("buckets", bucket.ID.String(), file.Path, file.Name),
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

func (s BucketService) DeleteFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	if file.Type == models.FileTypeFolder {
		return s.TrashFolder(logger, user, file)
	}
	return s.TrashFile(logger, user, file)
}

func (s BucketService) DownloadFile(
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

	url, err := s.Storage.PresignedGetObject(
		path.Join("buckets", file.BucketID.String(), file.Path, file.Name),
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

func (s BucketService) GetActivity(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) []map[string]interface{} {
	buckets := s.GetBucketList(logger, user, ids)

	var bucketIDs []string
	for _, bucket := range buckets {
		bucketIDs = append(bucketIDs, bucket.ID.String())
	}

	if len(bucketIDs) > 0 {
		searchCriteria := map[string][]string{
			"domain":      {c.DefaultDomain},
			"object_type": {rbac.ResourceBucket.String(), rbac.ResourceFile.String()},
			"bucket_id":   bucketIDs,
		}

		history, err := s.ActivityLogger.Search(searchCriteria)
		if err != nil {
			logger.Error("Search history failed", zap.Error(err))
			return []map[string]interface{}{}
		}

		if len(history) == 0 {
			return []map[string]interface{}{}
		}

		return activity.EnrichActivity(s.DB, history)
	}

	return []map[string]interface{}{}
}

func (s BucketService) GetBucketActivity(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) (models.Page[map[string]interface{}], error) {
	bucket, err := s.GetBucket(logger, user, ids)
	if err != nil {
		return models.Page[map[string]interface{}]{}, err
	}

	searchCriteria := map[string][]string{
		"domain":      {c.DefaultDomain},
		"object_type": {rbac.ResourceBucket.String(), rbac.ResourceFile.String()},
		"bucket_id":   {bucket.ID.String()},
	}

	history, err := s.ActivityLogger.Search(searchCriteria)
	if err != nil {
		logger.Error("Search history failed", zap.Error(err))
		return models.Page[map[string]interface{}]{}, err
	}

	if len(history) == 0 {
		return models.Page[map[string]interface{}]{}, nil
	}

	enriched := activity.EnrichActivity(s.DB, history)

	return models.Page[map[string]interface{}]{Data: enriched}, nil
}

// TrashFile moves a file to trash (soft delete).
func (s BucketService) TrashFile(logger *zap.Logger, user models.UserClaims, file models.File) error {
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

		objectPath := path.Join("buckets", file.BucketID.String(), file.Path, file.Name)

		if err := s.Storage.MarkFileAsTrashed(objectPath, models.TrashMetadata{
			OriginalPath: objectPath,
			TrashedAt:    now,
			TrashedBy:    user.UserID,
			FileID:       file.ID,
			IsFolder:     false,
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

// RestoreFile recovers a file or folder from trash.
func (s BucketService) RestoreFile(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	if file.Type == models.FileTypeFolder {
		return s.RestoreFolder(logger, user, file)
	}
	if file.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFileNotInTrash)
	}
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	if file.TrashedAt != nil && time.Since(*file.TrashedAt) > retentionPeriod {
		return errors.NewAPIError(410, errors.ErrFileTrashExpired)
	}

	// Check for naming conflicts
	var existingFile models.File
	result := s.DB.Where(
		"bucket_id = ? AND name = ? AND path = ? AND status != ?",
		bucketID, file.Name, file.Path, models.FileStatusTrashed,
	).First(&existingFile)

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

		objectPath := path.Join("buckets", bucketID.String(), file.Path, file.Name)
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

// ListTrashedFiles returns all trashed files for a bucket within 7-day window.
func (s BucketService) ListTrashedFiles(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.File {
	var files []models.File
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	cutoffDate := time.Now().Add(-retentionPeriod)
	result := s.DB.
		Preload("TrashedUser").
		Where(
			"bucket_id = ? AND status = ? AND trashed_at > ?",
			ids[0],
			models.FileStatusTrashed,
			cutoffDate,
		).
		Order("trashed_at DESC").
		Find(&files)

	if result.Error != nil {
		logger.Error("Failed to list trashed files", zap.Error(result.Error))
		return []models.File{}
	}

	return files
}

// PurgeFile permanently deletes a file or folder from trash (hard delete).
func (s BucketService) PurgeFile(logger *zap.Logger, user models.UserClaims, ids uuid.UUIDs) error {
	bucketID, fileID := ids[0], ids[1]

	file, err := sql.GetFileByID(s.DB, bucketID, fileID)
	if err != nil {
		return err
	}

	// Dispatch to folder purge if it's a folder
	if file.Type == models.FileTypeFolder {
		return s.PurgeFolder(logger, user, file)
	}

	// Validate file is in trash
	if file.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFileNotInTrash)
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		objectPath := path.Join("buckets", bucketID.String(), file.Path, file.Name)

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

// TrashFolder moves a folder and all its contents to trash (async).
func (s BucketService) TrashFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.File,
) error {
	// Validate it's actually a folder
	if folder.Type != models.FileTypeFolder {
		return errors.NewAPIError(400, errors.ErrNotAFolder)
	}

	// Don't allow trashing already-trashed folders
	if folder.Status == models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFolderAlreadyTrashed)
	}

	now := time.Now()

	// Update folder to trashed status immediately
	updates := map[string]interface{}{
		"status":     models.FileStatusTrashed,
		"trashed_at": now,
		"trashed_by": user.UserID,
	}

	if err := s.DB.Model(&folder).Updates(updates).Error; err != nil {
		logger.Error("Failed to update folder status to trashed", zap.Error(err))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	// Trigger async trash event to handle children
	event := events.NewFolderTrash(s.Publisher, folder.BucketID, folder.ID, user.UserID, now)
	event.Trigger()

	logger.Info("Folder trash initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

// RestoreFolder recovers a folder and all its contents from trash (async).
func (s BucketService) RestoreFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.File,
) error {
	// Validate it's actually a folder
	if folder.Type != models.FileTypeFolder {
		return errors.NewAPIError(400, errors.ErrNotAFolder)
	}

	// Validate folder is in trash
	if folder.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFolderNotInTrash)
	}

	// Check if expired (extra safety check)
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	if folder.TrashedAt != nil && time.Since(*folder.TrashedAt) > retentionPeriod {
		return errors.NewAPIError(410, errors.ErrFolderTrashExpired)
	}

	// Check for naming conflicts at the folder level
	var existingFolder models.File
	result := s.DB.Where(
		"bucket_id = ? AND name = ? AND path = ? AND type = 'folder' AND status != ? AND status != ?",
		folder.BucketID,
		folder.Name,
		folder.Path,
		models.FileStatusTrashed,
		models.FileStatusRestoring,
	).First(&existingFolder)

	if result.RowsAffected > 0 {
		return errors.NewAPIError(409, errors.ErrFolderNameConflict)
	}

	// Set folder to restoring status
	if err := s.DB.Model(&folder).Update("status", models.FileStatusRestoring).Error; err != nil {
		logger.Error("Failed to set folder to restoring status", zap.Error(err))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	// Trigger async restore event
	event := events.NewFolderRestore(s.Publisher, folder.BucketID, folder.ID, user.UserID)
	event.Trigger()

	logger.Info("Folder restore initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

// PurgeFolder permanently deletes a folder and all its contents from trash (async).
func (s BucketService) PurgeFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.File,
) error {
	// Validate it's actually a folder
	if folder.Type != models.FileTypeFolder {
		return errors.NewAPIError(400, errors.ErrNotAFolder)
	}

	// Validate folder is in trash
	if folder.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFolderNotInTrash)
	}

	// Trigger async purge event
	event := events.NewFolderPurge(s.Publisher, folder.BucketID, folder.ID, user.UserID)
	event.Trigger()

	logger.Info("Folder purge initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}
