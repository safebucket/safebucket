package sql

import (
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func LockBucketForUpdate(tx *gorm.DB, bucketID uuid.UUID) error {
	var bucket models.Bucket
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		Where("id = ?", bucketID).
		First(&bucket).
		Error
}

func LockMoveTarget(tx *gorm.DB, bucketID uuid.UUID, targetID *uuid.UUID) (*models.Folder, error) {
	if targetID == nil {
		return nil, nil
	}

	var target models.Folder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND bucket_id = ?", *targetID, bucketID).
		First(&target).Error; err != nil {
		return nil, err
	}
	return &target, nil
}

func LockFilesForMove(tx *gorm.DB, bucketID uuid.UUID, ids uuid.UUIDs) ([]models.File, error) {
	if len(ids) == 0 {
		return []models.File{}, nil
	}

	files := make([]models.File, 0, len(ids))
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND id IN ?", bucketID, ids).
		Find(&files).
		Error
	return files, err
}

func LockFoldersForMove(tx *gorm.DB, bucketID uuid.UUID, ids uuid.UUIDs) ([]models.Folder, error) {
	if len(ids) == 0 {
		return []models.Folder{}, nil
	}

	folders := make([]models.Folder, 0, len(ids))
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bucket_id = ? AND id IN ?", bucketID, ids).
		Find(&folders).
		Error
	return folders, err
}

func IsInvalidMoveTarget(
	tx *gorm.DB,
	bucketID uuid.UUID,
	targetID *uuid.UUID,
	folderIDs uuid.UUIDs,
) (bool, error) {
	if targetID == nil || len(folderIDs) == 0 {
		return false, nil
	}

	var count int64
	err := tx.Raw(`
		WITH RECURSIVE folder_ancestors (id, folder_id) AS (
			SELECT id, folder_id
			FROM folders
			WHERE id = ? AND bucket_id = ? AND deleted_at IS NULL
			UNION
			SELECT parent.id, parent.folder_id
			FROM folders AS parent
			JOIN folder_ancestors AS child ON parent.id = child.folder_id
			WHERE parent.bucket_id = ? AND parent.deleted_at IS NULL
		)
		SELECT COUNT(*) FROM folder_ancestors WHERE id IN ?
	`, *targetID, bucketID, bucketID, folderIDs).Scan(&count).Error
	return count > 0, err
}

func HasSameFileNamesAtTarget(
	tx *gorm.DB,
	bucketID uuid.UUID,
	targetID *uuid.UUID,
	names []string,
) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}

	query := tx.Model(&models.File{}).Where("bucket_id = ? AND name IN ?", bucketID, names)
	query = inFolder(query, targetID)
	var file models.File
	result := query.Select("id").Limit(1).Find(&file)
	return result.RowsAffected > 0, result.Error
}

func HasSameFolderNamesAtTarget(
	tx *gorm.DB,
	bucketID uuid.UUID,
	targetID *uuid.UUID,
	names []string,
) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}

	query := tx.Model(&models.Folder{}).Where("bucket_id = ? AND name IN ?", bucketID, names)
	query = inFolder(query, targetID)
	var folder models.Folder
	result := query.Select("id").Limit(1).Find(&folder)
	return result.RowsAffected > 0, result.Error
}

func MoveFiles(tx *gorm.DB, bucketID uuid.UUID, ids uuid.UUIDs, targetID *uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := tx.Model(&models.File{}).
		Where("bucket_id = ? AND id IN ?", bucketID, ids).
		Update("folder_id", targetID)
	return result.RowsAffected, result.Error
}

func MoveFolders(tx *gorm.DB, bucketID uuid.UUID, ids uuid.UUIDs, targetID *uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := tx.Model(&models.Folder{}).
		Where("bucket_id = ? AND id IN ?", bucketID, ids).
		Update("folder_id", targetID)
	return result.RowsAffected, result.Error
}

func inFolder(query *gorm.DB, folderID *uuid.UUID) *gorm.DB {
	if folderID == nil {
		return query.Where("folder_id IS NULL")
	}
	return query.Where("folder_id = ?", *folderID)
}
