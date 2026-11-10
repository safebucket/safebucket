package sql

import (
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RestoreChildFolders(
	tx *gorm.DB,
	bucketID, parentID uuid.UUID,
	before time.Time,
	limit int,
) ([]models.Folder, error) {
	var candidates []models.Folder
	err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND folder_id = ? AND status = ? AND deleted_at < ?",
			bucketID, parentID, models.FolderStatusDeleted, before).
		Where(`NOT EXISTS (
			SELECT 1 FROM folders AS f
			WHERE f.bucket_id = folders.bucket_id AND f.folder_id = folders.folder_id
			  AND f.name = folders.name AND f.deleted_at IS NULL
		)`).
		Order("deleted_at DESC").
		Limit(limit).
		Find(&candidates).Error

	if err != nil {
		return nil, err
	}

	restored := make([]models.Folder, 0, len(candidates))
	ids := make([]uuid.UUID, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, folder := range candidates {
		if !seen[folder.Name] {
			seen[folder.Name] = true
			restored = append(restored, folder)
			ids = append(ids, folder.ID)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	err = tx.Unscoped().Model(&models.Folder{}).Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     models.FolderStatusCreated,
			"deleted_at": nil,
			"deleted_by": nil,
			"updated_at": time.Now(),
		}).Error
	return restored, err
}

func RestoreChildFiles(
	tx *gorm.DB,
	bucketID, parentID uuid.UUID,
	before time.Time,
	limit int,
) ([]models.File, error) {
	var candidates []models.File
	err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND folder_id = ? AND (status IS NULL OR status = ?) AND deleted_at < ?",
			bucketID, parentID, models.FileStatusDeleted, before).
		Where(`NOT EXISTS (
			SELECT 1 FROM files AS f
			WHERE f.bucket_id = files.bucket_id AND f.folder_id = files.folder_id
			  AND f.name = files.name AND f.deleted_at IS NULL
		)`).
		Order("deleted_at DESC").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}

	restored := make([]models.File, 0, len(candidates))
	var uploadedIDs, uploadingIDs []uuid.UUID
	seen := make(map[string]bool, len(candidates))
	for _, file := range candidates {
		if seen[file.Name] {
			continue
		}
		seen[file.Name] = true
		restored = append(restored, file)
		if file.CurrentVersionID == nil {
			uploadingIDs = append(uploadingIDs, file.ID)
		} else {
			uploadedIDs = append(uploadedIDs, file.ID)
		}
	}

	if err = restoreFiles(tx, uploadedIDs, models.FileStatusUploaded); err != nil {
		return nil, err
	}
	if err = restoreFiles(tx, uploadingIDs, models.FileStatusUploading); err != nil {
		return nil, err
	}
	return restored, nil
}

func restoreFiles(tx *gorm.DB, ids []uuid.UUID, status models.FileStatus) error {
	if len(ids) == 0 {
		return nil
	}

	return tx.Unscoped().
		Model(&models.File{}).
		Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     status,
			"deleted_at": nil,
			"deleted_by": nil,
			"updated_at": time.Now(),
		}).Error
}
