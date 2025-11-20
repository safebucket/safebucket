package services

import (
	"path"
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
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketFolderService struct {
	DB                 *gorm.DB
	Storage            storage.IStorage
	Publisher          messaging.IPublisher
	ActivityLogger     activity.IActivityLogger
	TrashRetentionDays int
}

func (s BucketFolderService) Routes() chi.Router {
	r := chi.NewRouter()

	// Create folder endpoint is mounted on bucket route
	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FolderCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateFolder))

	r.Route("/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteFolder))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FolderUpdateBody]).
			Patch("/", handlers.UpdateHandler(s.UpdateFolder))

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Post("/restore", handlers.DeleteHandler(s.RestoreFolder))
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
		return models.Folder{}, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	// Check if parent folder exists (if folder_id is provided)
	if body.FolderID != nil {
		var parentFolder models.Folder
		result = s.DB.Where("id = ? AND bucket_id = ?", body.FolderID, bucketID).Find(&parentFolder)
		if result.RowsAffected == 0 {
			return models.Folder{}, errors.NewAPIError(404, "PARENT_FOLDER_NOT_FOUND")
		}
	}

	// Check for duplicate folder name in the same parent
	var existingFolder models.Folder
	query := s.DB.Where("bucket_id = ? AND name = ?", bucketID, body.Name)
	if body.FolderID != nil {
		query = query.Where("folder_id = ?", body.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFolder)
	if result.RowsAffected > 0 {
		return models.Folder{}, errors.NewAPIError(409, "FOLDER_ALREADY_EXISTS")
	}

	folder := models.Folder{
		Name:     body.Name,
		BucketID: bucketID,
		FolderID: body.FolderID,
	}

	if err := s.DB.Create(&folder).Error; err != nil {
		logger.Error("Failed to create folder", zap.Error(err))
		return models.Folder{}, errors.ErrCreateFailed
	}

	action := models.Activity{
		Message: activity.FolderCreated,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionCreate.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folder.ID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log folder creation activity", zap.Error(err))
	}

	return folder, nil
}

func (s BucketFolderService) UpdateFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FolderUpdateBody,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
	}

	// Check for duplicate folder name in the same parent (excluding current folder)
	var existingFolder models.Folder
	query := s.DB.Where("bucket_id = ? AND name = ? AND id != ?", bucketID, body.Name, folderID)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.Find(&existingFolder)
	if result.RowsAffected > 0 {
		return errors.NewAPIError(409, "FOLDER_NAME_CONFLICT")
	}

	folder.Name = body.Name
	if err := s.DB.Save(&folder).Error; err != nil {
		logger.Error("Failed to update folder", zap.Error(err))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	action := models.Activity{
		Message: activity.FolderUpdated,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionUpdate.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folderID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log folder update activity", zap.Error(err))
	}

	return nil
}

func (s BucketFolderService) DeleteFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
	}

	return s.TrashFolder(logger, user, folder)
}

// TrashFolder moves a folder and all its contents to trash (async).
func (s BucketFolderService) TrashFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
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

	// Create trash marker for folder
	objectPath := path.Join("folder", folder.BucketID.String(), folder.ID.String())
	if err := s.Storage.MarkFileAsTrashed(objectPath, models.TrashMetadata{
		TrashedAt: now,
		TrashedBy: user.UserID,
		ObjectID:  folder.ID,
		IsFolder:  true,
	}); err != nil {
		logger.Warn("Failed to create trash marker for folder", zap.Error(err))
		// Continue - folder exists only in database
	}

	// Trigger async trash event to handle children
	event := events.NewFolderTrash(s.Publisher, folder.BucketID, folder.ID, user.UserID, now)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderTrashed,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionErase.String(),
			"bucket_id":   folder.BucketID.String(),
			"folder_id":   folder.ID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log trash activity", zap.Error(err))
	}

	logger.Info("Folder trash initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

// RestoreFolder recovers a folder and all its contents from trash (async).
func (s BucketFolderService) RestoreFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
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
	var existingFolder models.Folder
	query := s.DB.Where(
		"bucket_id = ? AND name = ? AND status != ? AND status != ?",
		bucketID,
		folder.Name,
		models.FileStatusTrashed,
		models.FileStatusRestoring,
	)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result = query.First(&existingFolder)

	if result.RowsAffected > 0 {
		return errors.NewAPIError(409, errors.ErrFolderNameConflict)
	}

	// Set folder to restoring status
	if err := s.DB.Model(&folder).Update("status", models.FileStatusRestoring).Error; err != nil {
		logger.Error("Failed to set folder to restoring status", zap.Error(err))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	// Remove trash marker
	objectPath := path.Join("folder", bucketID.String(), folderID.String())
	if err := s.Storage.UnmarkFileAsTrashed(objectPath); err != nil {
		logger.Warn("Failed to remove trash marker for folder", zap.Error(err))
		// Continue - marker might not exist or already removed
	}

	// Trigger async restore event
	event := events.NewFolderRestore(s.Publisher, bucketID, folderID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderRestored,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folderID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log restore activity", zap.Error(err))
	}

	logger.Info("Folder restore initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

// PurgeFolder permanently deletes a folder and all its contents from trash (async).
func (s BucketFolderService) PurgeFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
	}

	// Validate folder is in trash
	if folder.Status != models.FileStatusTrashed {
		return errors.NewAPIError(400, errors.ErrFolderNotInTrash)
	}

	// Trigger async purge event
	event := events.NewFolderPurge(s.Publisher, bucketID, folderID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderPurged,
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionPurge.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folderID.String(),
			"domain":      c.DefaultDomain,
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log purge activity", zap.Error(err))
	}

	logger.Info("Folder purge initiated (async)",
		zap.String("folder", folder.Name),
		zap.String("folder_id", folder.ID.String()))

	return nil
}

// ListTrashedFolders returns all trashed folders for a bucket within retention window.
func (s BucketFolderService) ListTrashedFolders(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.Folder {
	var folders []models.Folder
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
		Find(&folders)

	if result.Error != nil {
		logger.Error("Failed to list trashed folders", zap.Error(result.Error))
		return []models.Folder{}
	}

	return folders
}
