package sql

import (
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func RestoreTreeFolders(
	tx *gorm.DB,
	bucketID, rootID uuid.UUID,
	before time.Time,
	limit int,
) ([]models.Folder, error) {
	var folders []models.Folder
	err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND status = ? AND deleted_at < ?",
			bucketID, models.FolderStatusDeleted, before).
		Where(`folder_id IN (
			WITH RECURSIVE live_tree (id) AS (
				SELECT id FROM folders WHERE id = ?
				UNION
				SELECT child.id
				FROM folders AS child
				JOIN live_tree AS parent ON child.folder_id = parent.id
				WHERE child.deleted_at IS NULL AND child.status = ?
			)
			SELECT id FROM live_tree
		)`, rootID, models.FolderStatusCreated).
		Where(`NOT EXISTS (
			SELECT 1 FROM folders AS f
			WHERE f.bucket_id = folders.bucket_id AND f.folder_id = folders.folder_id
			  AND f.name = folders.name
			  AND (f.deleted_at IS NULL OR (f.status = ? AND f.deleted_at < ? AND f.deleted_at > folders.deleted_at))
		)`, models.FolderStatusDeleted, before).
		Order("deleted_at DESC").
		Limit(limit).
		Find(&folders).Error
	if err != nil {
		return nil, err
	}

	if len(folders) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(folders))
	for _, folder := range folders {
		ids = append(ids, folder.ID)
	}

	err = tx.Unscoped().Model(&models.Folder{}).Where("id IN ?", ids).
		Updates(map[string]any{
			"status":     models.FolderStatusCreated,
			"deleted_at": nil,
			"deleted_by": nil,
			"updated_at": time.Now(),
		}).Error
	return folders, err
}

func RestoreTreeFiles(
	tx *gorm.DB,
	bucketID, rootID uuid.UUID,
	before time.Time,
	limit int,
) ([]models.File, error) {
	var files []models.File
	err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND (status IS NULL OR status = ?) AND deleted_at < ?",
			bucketID, models.FileStatusDeleted, before).
		Where(`folder_id IN (
			WITH RECURSIVE live_tree (id) AS (
				SELECT id FROM folders WHERE id = ?
				UNION
				SELECT child.id
				FROM folders AS child
				JOIN live_tree AS parent ON child.folder_id = parent.id
				WHERE child.deleted_at IS NULL AND child.status = ?
			)
			SELECT id FROM live_tree
		)`, rootID, models.FolderStatusCreated).
		Where(`NOT EXISTS (
			SELECT 1 FROM files AS f
			WHERE f.bucket_id = files.bucket_id AND f.folder_id = files.folder_id
			  AND f.name = files.name
			  AND (f.deleted_at IS NULL OR ((f.status IS NULL OR f.status = ?) AND f.deleted_at < ? AND f.deleted_at > files.deleted_at))
		)`, models.FileStatusDeleted, before).
		Order("deleted_at DESC").
		Limit(limit).
		Find(&files).Error
	if err != nil {
		return nil, err
	}

	var uploadedIDs, uploadingIDs []uuid.UUID
	for _, file := range files {
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
	return files, nil
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

func TreeHasDeletingItems(tx *gorm.DB, rootID uuid.UUID) (bool, error) {
	var found bool
	err := tx.Raw(`
		WITH RECURSIVE folder_tree (id) AS (
			SELECT id FROM folders WHERE id = ?
			UNION
			SELECT child.id
			FROM folders AS child
			JOIN folder_tree AS parent ON child.folder_id = parent.id
		)
		SELECT EXISTS (SELECT 1 FROM folders WHERE folder_id IN (SELECT id FROM folder_tree) AND status = ?)
		    OR EXISTS (SELECT 1 FROM files WHERE folder_id IN (SELECT id FROM folder_tree) AND status = ?)`,
		rootID, models.FolderStatusDeleting, models.FileStatusDeleting,
	).Scan(&found).Error
	return found, err
}

func TreeHasRestoringFolders(tx *gorm.DB, rootIDs uuid.UUIDs) (bool, error) {
	var found bool
	err := tx.Raw(`
		WITH RECURSIVE folder_tree (id) AS (
			SELECT id FROM folders WHERE id IN ?
			UNION
			SELECT child.id
			FROM folders AS child
			JOIN folder_tree AS parent ON child.folder_id = parent.id
		)
		SELECT EXISTS (SELECT 1 FROM folders WHERE id IN (SELECT id FROM folder_tree) AND status = ?)`,
		rootIDs, models.FolderStatusRestoring,
	).Scan(&found).Error
	return found, err
}
