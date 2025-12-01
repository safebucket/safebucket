package services

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"time"

	"api/internal/activity"
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

	r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
		With(m.Validate[models.FolderCreateBody]).
		Post("/", handlers.CreateHandler(s.CreateFolder))

	r.Route("/{id1}", func(r chi.Router) {
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Patch("/", s.patchFolderHandler)

		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			Delete("/", handlers.DeleteHandler(s.DeleteFolder))
	})

	return r
}

// patchFolderHandler routes between name update and trash operations based on request body.
func (s BucketFolderService) patchFolderHandler(w http.ResponseWriter, r *http.Request) {
	// Try to parse as FolderPatchBody first
	var patchBody models.FolderPatchBody
	if err := json.NewDecoder(r.Body).Decode(&patchBody); err == nil && patchBody.Status != "" {
		// Reset body for validation middleware
		bodyBytes, _ := json.Marshal(patchBody)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Use trash operation handler
		handler := m.Validate[models.FolderPatchBody](
			handlers.UpdateHandler(s.PatchFolder),
		)
		handler.ServeHTTP(w, r)
		return
	}

	// Otherwise, treat as FolderUpdateBody (name change)
	handler := m.Validate[models.FolderUpdateBody](
		handlers.UpdateHandler(s.UpdateFolder),
	)
	handler.ServeHTTP(w, r)
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
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionCreate.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folder.ID.String(),
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
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionUpdate.String(),
			"bucket_id":   bucketID.String(),
			"folder_id":   folderID.String(),
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log folder update activity", zap.Error(err))
	}

	return nil
}

// PatchFolder handles PATCH requests for trash/restore operations on folders.
func (s BucketFolderService) PatchFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.FolderPatchBody,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Unscoped().Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
	}

	switch body.Status {
	case models.FileStatusTrashed:
		return s.TrashFolder(logger, user, folder)
	case models.FileStatusUploaded:
		return s.RestoreFolder(logger, user, folder)
	case models.FileStatusUploading, models.FileStatusDeleting, models.FileStatusRestoring, models.FileStatusDeleted:
		return errors.NewAPIError(400, "INVALID_STATUS")
	default:
		return errors.NewAPIError(400, "INVALID_STATUS")
	}
}

// DeleteFolder handles DELETE requests for permanent folder deletion (purge).
func (s BucketFolderService) DeleteFolder(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
) error {
	bucketID, folderID := ids[0], ids[1]

	var folder models.Folder
	result := s.DB.Unscoped().Where("id = ? AND bucket_id = ?", folderID, bucketID).First(&folder)
	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
	}

	return s.PurgeFolder(logger, user, folder)
}

// TrashFolder moves a folder and all its contents to trash (async) with atomic status transition.
func (s BucketFolderService) TrashFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	// Update folder to trashed status and set trashed_by for audit trail
	updates := map[string]interface{}{
		"status":     models.FileStatusTrashed,
		"trashed_by": user.UserID,
	}
	result := s.DB.Model(&folder).
		Where("status IS NULL OR status = ?", models.FileStatusUploaded).
		Updates(updates)

	if result.Error != nil {
		logger.Error("Failed to update folder status to trashed", zap.Error(result.Error))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	// Check if any rows were updated (atomic conflict detection)
	if result.RowsAffected == 0 {
		// Re-fetch to get current status
		var currentFolder models.Folder
		fetchResult := s.DB.Unscoped().Where("id = ?", folder.ID).Find(&currentFolder)
		if fetchResult.RowsAffected > 0 {
			if currentFolder.Status == models.FileStatusTrashed {
				return errors.NewAPIError(409, "FOLDER_ALREADY_TRASHED")
			}
			if currentFolder.Status == models.FileStatusRestoring {
				return errors.NewAPIError(409, "FOLDER_RESTORE_IN_PROGRESS")
			}
		}
		return errors.NewAPIError(409, "INVALID_FOLDER_STATUS_TRANSITION")
	}

	// Soft delete folder using GORM (sets deleted_at)
	if err := s.DB.Delete(&folder).Error; err != nil {
		logger.Error("Failed to soft delete folder", zap.Error(err))
		return errors.NewAPIError(500, "DELETE_FAILED")
	}

	// Create trash marker for folder
	objectPath := path.Join("buckets", folder.BucketID.String(), folder.ID.String())
	if err := s.Storage.MarkAsTrashed(objectPath, folder); err != nil {
		logger.Warn("Failed to create trash marker for folder", zap.Error(err))
		// Continue - folder exists only in database
	}

	// Trigger async trash event to handle children
	event := events.NewFolderTrash(s.Publisher, folder.BucketID, folder.ID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderTrashed,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionErase.String(),
			"bucket_id":   folder.BucketID.String(),
			"folder_id":   folder.ID.String(),
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

// RestoreFolder recovers a folder and all its contents from trash (async) with atomic status transition.
func (s BucketFolderService) RestoreFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	// Check if expired (extra safety check)
	retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
	if folder.DeletedAt.Valid && time.Since(folder.DeletedAt.Time) > retentionPeriod {
		return errors.NewAPIError(410, errors.ErrFolderTrashExpired)
	}

	// Check for naming conflicts at the folder level
	var existingFolder models.Folder
	query := s.DB.Where(
		"bucket_id = ? AND name = ? AND status != ? AND status != ?",
		folder.BucketID,
		folder.Name,
		models.FileStatusTrashed,
		models.FileStatusRestoring,
	)
	if folder.FolderID != nil {
		query = query.Where("folder_id = ?", folder.FolderID)
	} else {
		query = query.Where("folder_id IS NULL")
	}
	result := query.Find(&existingFolder)

	if result.RowsAffected > 0 {
		return errors.NewAPIError(409, errors.ErrFolderNameConflict)
	}

	// Set folder to restoring status with atomic transition
	result = s.DB.Unscoped().Model(&folder).
		Where("status = ?", models.FileStatusTrashed).
		Update("status", models.FileStatusRestoring)

	if result.Error != nil {
		logger.Error("Failed to set folder to restoring status", zap.Error(result.Error))
		return errors.NewAPIError(500, "UPDATE_FAILED")
	}

	// Check if any rows were updated (atomic conflict detection)
	if result.RowsAffected == 0 {
		// Re-fetch to get current status
		var currentFolder models.Folder
		fetchResult := s.DB.Unscoped().Where("id = ?", folder.ID).Find(&currentFolder)
		if fetchResult.RowsAffected > 0 {
			if currentFolder.Status == models.FileStatusRestoring {
				return errors.NewAPIError(409, "FOLDER_RESTORE_IN_PROGRESS")
			}
			if currentFolder.Status != models.FileStatusTrashed {
				return errors.NewAPIError(409, "FOLDER_NOT_IN_TRASH")
			}
		}
		return errors.NewAPIError(409, "INVALID_FOLDER_STATUS_TRANSITION")
	}

	// Remove trash marker
	objectPath := path.Join("buckets", folder.BucketID.String(), folder.ID.String())
	if err := s.Storage.UnmarkAsTrashed(objectPath, folder); err != nil {
		logger.Warn("Failed to remove trash marker for folder", zap.Error(err))
		// Continue - marker might not exist or already removed
	}

	// Trigger async restore event
	event := events.NewFolderRestore(s.Publisher, folder.BucketID, folder.ID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderRestored,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   folder.BucketID.String(),
			"folder_id":   folder.ID.String(),
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

// PurgeFolder permanently deletes a folder and all its contents from trash (async) with atomic status check.
func (s BucketFolderService) PurgeFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	// Atomic status check: only allow purging trashed folders
	var currentFolder models.Folder
	result := s.DB.Unscoped().Where("id = ? AND bucket_id = ? AND status = ?",
		folder.ID, folder.BucketID, models.FileStatusTrashed).
		First(&currentFolder)

	if result.Error != nil || result.RowsAffected == 0 {
		return errors.NewAPIError(409, "FOLDER_NOT_IN_TRASH")
	}

	// Trigger async purge event
	event := events.NewFolderPurge(s.Publisher, folder.BucketID, folder.ID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderPurged,
		Object:  folder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionPurge.String(),
			"bucket_id":   folder.BucketID.String(),
			"folder_id":   folder.ID.String(),
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
	// Use Unscoped to query soft-deleted (trashed) folders
	result := s.DB.Unscoped().
		Where(
			"bucket_id = ? AND status = ? AND deleted_at IS NOT NULL",
			ids[0],
			models.FileStatusTrashed,
		).
		Order("deleted_at DESC").
		Find(&folders)

	if result.Error != nil {
		logger.Error("Failed to list trashed folders", zap.Error(result.Error))
		return []models.Folder{}
	}

	return folders
}
