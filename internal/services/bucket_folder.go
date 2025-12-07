package services

import (
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
	"gorm.io/gorm/clause"
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
		// PUT for name updates (RESTful full resource update)
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FolderUpdateBody]).
			Put("/", handlers.UpdateHandler(s.UpdateFolder))

		// PATCH for status updates (trash/restore) - consistent with files
		r.With(m.AuthorizeGroup(s.DB, models.GroupContributor, 0)).
			With(m.Validate[models.FolderPatchBody]).
			Patch("/", handlers.UpdateHandler(s.PatchFolder))

		// DELETE for permanent deletion
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
	case "deleted":
		return s.TrashFolder(logger, user, folder)
	case "uploaded":
		return s.RestoreFolder(logger, user, folder)
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
	// Check that folder is not already soft-deleted
	if folder.DeletedAt.Valid {
		return errors.NewAPIError(409, "FOLDER_ALREADY_TRASHED")
	}

	// Check if folder is in restoring state
	if folder.Status == models.FileStatusRestoring {
		return errors.NewAPIError(409, "FOLDER_RESTORE_IN_PROGRESS")
	}

	// Update status to deleted and set deleted_by for audit trail
	updates := map[string]interface{}{
		"status":     models.FileStatusDeleted,
		"deleted_by": user.UserID,
	}
	if err := s.DB.Model(&folder).Updates(updates).Error; err != nil {
		logger.Error("Failed to update folder for trashing", zap.Error(err))
		return errors.NewAPIError(500, "UPDATE_FAILED")
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

// restoreParentFolders restores all trashed parent folders in the hierarchy.
// It traverses from the given folder up to the root, collecting trashed folders,
// then restores them from root to leaf to avoid naming conflicts.
// Returns the list of restored folders so their trash markers can be removed
// AFTER the database updates are committed.
func (s BucketFolderService) restoreParentFolders(
	tx *gorm.DB,
	logger *zap.Logger,
	folderID *uuid.UUID,
	bucketID uuid.UUID,
) ([]models.Folder, error) {
	if folderID == nil {
		return nil, nil
	}

	// Collect all trashed parent folder IDs first (from immediate parent to root)
	var trashedFolderIDs []uuid.UUID
	currentFolderID := folderID

	for currentFolderID != nil {
		var folder models.Folder
		result := tx.Unscoped().Where("id = ? AND bucket_id = ?", currentFolderID, bucketID).First(&folder)
		if result.Error != nil {
			break // Parent folder doesn't exist, stop traversal
		}

		// If folder is trashed (soft-deleted), add to list
		if folder.DeletedAt.Valid {
			trashedFolderIDs = append(trashedFolderIDs, folder.ID)
		}

		currentFolderID = folder.FolderID
	}

	if len(trashedFolderIDs) == 0 {
		return nil, nil // No trashed parent folders
	}

	// Now lock and fetch all trashed folders with FOR UPDATE to prevent concurrent modifications
	var trashedFolders []models.Folder
	result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ?", trashedFolderIDs).
		Find(&trashedFolders)
	if result.Error != nil {
		return nil, result.Error
	}

	// Build a map for quick lookup and maintain order (root to leaf)
	folderMap := make(map[uuid.UUID]models.Folder)
	for _, f := range trashedFolders {
		folderMap[f.ID] = f
	}

	var restoredFolders []models.Folder

	// Restore folders from root to leaf (reverse order of trashedFolderIDs)
	// This ensures parent folders exist before restoring child folders
	for i := len(trashedFolderIDs) - 1; i >= 0; i-- {
		folder, exists := folderMap[trashedFolderIDs[i]]
		if !exists {
			continue // Folder was deleted between initial scan and lock
		}

		// Skip if already being restored or no longer trashed
		if folder.Status == models.FileStatusRestoring || !folder.DeletedAt.Valid {
			continue
		}

		// Check for naming conflicts (only against active folders)
		var existingFolder models.Folder
		query := tx.Where(
			"bucket_id = ? AND name = ? AND id != ?",
			folder.BucketID, folder.Name, folder.ID,
		)
		if folder.FolderID != nil {
			query = query.Where("folder_id = ?", folder.FolderID)
		} else {
			query = query.Where("folder_id IS NULL")
		}
		if query.Find(&existingFolder); existingFolder.ID != uuid.Nil {
			return nil, errors.NewAPIError(409, "PARENT_FOLDER_NAME_CONFLICT")
		}

		// Restore the folder in database
		updates := map[string]interface{}{
			"deleted_at": nil,
			"deleted_by": nil,
			"status":     nil,
		}
		if err := tx.Unscoped().Model(&folder).Updates(updates).Error; err != nil {
			logger.Error("Failed to restore parent folder",
				zap.Error(err),
				zap.String("folder_id", folder.ID.String()))
			return nil, err
		}

		// Add to list of restored folders (will unmark from storage after DB commit)
		restoredFolders = append(restoredFolders, folder)

		logger.Info("Restored parent folder",
			zap.String("folder_name", folder.Name),
			zap.String("folder_id", folder.ID.String()))
	}

	return restoredFolders, nil
}

// unmarkRestoredFolders removes trash markers for restored folders.
// This must be called AFTER the database updates to avoid race conditions
// with the trash expiration handler.
func (s BucketFolderService) unmarkRestoredFolders(logger *zap.Logger, folders []models.Folder) {
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

// RestoreFolder recovers a folder and all its contents from trash (async) with atomic status transition.
func (s BucketFolderService) RestoreFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	// Collect restored folders to unmark from storage after transaction commits
	var restoredParentFolders []models.Folder
	var restoredFolder models.Folder

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the folder row to prevent concurrent modifications
		var lockedFolder models.Folder
		result := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND bucket_id = ?", folder.ID, folder.BucketID).
			First(&lockedFolder)

		if result.Error != nil {
			return errors.NewAPIError(404, "FOLDER_NOT_FOUND")
		}

		// Re-check conditions after acquiring lock (state may have changed)
		if !lockedFolder.DeletedAt.Valid {
			return errors.NewAPIError(409, "FOLDER_NOT_IN_TRASH")
		}

		if lockedFolder.Status == models.FileStatusRestoring {
			return errors.NewAPIError(409, "FOLDER_RESTORE_IN_PROGRESS")
		}

		// Check if expired (extra safety check)
		retentionPeriod := time.Duration(s.TrashRetentionDays) * 24 * time.Hour
		if time.Since(lockedFolder.DeletedAt.Time) > retentionPeriod {
			return errors.NewAPIError(410, errors.ErrFolderTrashExpired)
		}

		// Restore parent folders if they are trashed (database only, defer storage unmark)
		parentFolders, err := s.restoreParentFolders(tx, logger, lockedFolder.FolderID, lockedFolder.BucketID)
		if err != nil {
			return err
		}
		restoredParentFolders = parentFolders

		// Check for naming conflicts at the folder level (only against active folders)
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
			return errors.NewAPIError(409, errors.ErrFolderNameConflict)
		}

		// Set folder to restoring status
		if err := tx.Unscoped().Model(&lockedFolder).Update("status", models.FileStatusRestoring).Error; err != nil {
			logger.Error("Failed to set folder to restoring status", zap.Error(err))
			return errors.NewAPIError(500, "UPDATE_FAILED")
		}

		// Store folder for unmarking after transaction commits
		restoredFolder = lockedFolder

		return nil
	})

	if err != nil {
		return err
	}

	// After transaction commits, unmark all restored items from storage
	// This ensures the trash expiration handler sees the committed state
	s.unmarkRestoredFolders(logger, restoredParentFolders)

	// Unmark this folder from storage
	objectPath := path.Join("buckets", restoredFolder.BucketID.String(), restoredFolder.ID.String())
	if err := s.Storage.UnmarkAsTrashed(objectPath, restoredFolder); err != nil {
		logger.Warn("Failed to remove trash marker for folder", zap.Error(err))
	}

	// Trigger async restore event
	event := events.NewFolderRestore(s.Publisher, restoredFolder.BucketID, restoredFolder.ID, user.UserID)
	event.Trigger()

	action := models.Activity{
		Message: activity.FolderRestored,
		Object:  restoredFolder.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      rbac.ActionRestore.String(),
			"bucket_id":   restoredFolder.BucketID.String(),
			"folder_id":   restoredFolder.ID.String(),
			"object_type": rbac.ResourceFolder.String(),
			"user_id":     user.UserID.String(),
		}),
	}

	if err := s.ActivityLogger.Send(action); err != nil {
		logger.Error("Failed to log restore activity", zap.Error(err))
	}

	logger.Info("Folder restore initiated (async)",
		zap.String("folder", restoredFolder.Name),
		zap.String("folder_id", restoredFolder.ID.String()))

	return nil
}

// PurgeFolder permanently deletes a folder and all its contents from trash (async) with atomic status check.
func (s BucketFolderService) PurgeFolder(
	logger *zap.Logger,
	user models.UserClaims,
	folder models.Folder,
) error {
	// Only allow purging soft-deleted folders (in trash)
	if !folder.DeletedAt.Valid {
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
	// Trashed = deleted_at IS NOT NULL and status != restoring
	result := s.DB.Unscoped().
		Where(
			"bucket_id = ? AND deleted_at IS NOT NULL AND (status IS NULL OR status != ?)",
			ids[0],
			models.FileStatusRestoring,
		).
		Order("deleted_at DESC").
		Find(&folders)

	if result.Error != nil {
		logger.Error("Failed to list trashed folders", zap.Error(result.Error))
		return []models.Folder{}
	}

	return folders
}
