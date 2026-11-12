package sql

import (
	"time"

	"github.com/safebucket/safebucket/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MarkExpiredFilesDeleting(tx *gorm.DB, expiredBefore time.Time, limit int) ([]models.File, error) {
	expired := tx.Unscoped().Model(&models.File{}).Select("files.id").
		Where("files.deleted_at < ? AND (files.status IS NULL OR files.status = ?)",
			expiredBefore, models.FileStatusDeleted).
		Where("EXISTS (SELECT 1 FROM buckets WHERE buckets.id = files.bucket_id AND buckets.deleted_at IS NULL)").
		Order("files.deleted_at, files.id").
		Limit(limit)

	now := time.Now()
	var files []models.File
	err := tx.Unscoped().Model(&files).Clauses(clause.Returning{}).
		Where("id IN (?) AND deleted_at < ? AND (status IS NULL OR status = ?)",
			expired, expiredBefore, models.FileStatusDeleted).
		Updates(map[string]any{
			"status":     models.FileStatusDeleting,
			"updated_at": now,
		}).Error
	return files, err
}

func DeleteExpiredEmptyFolders(tx *gorm.DB, expiredBefore time.Time, limit int) ([]models.Folder, error) {
	empty := tx.Unscoped().Model(&models.Folder{}).Select("folders.id").
		Where("folders.deleted_at < ? AND folders.status = ?",
			expiredBefore, models.FolderStatusDeleted).
		Where("NOT EXISTS (SELECT 1 FROM folders AS child WHERE child.folder_id = folders.id)").
		Where("NOT EXISTS (SELECT 1 FROM files WHERE files.folder_id = folders.id)").
		Where("EXISTS (SELECT 1 FROM buckets WHERE buckets.id = folders.bucket_id AND buckets.deleted_at IS NULL)").
		Order("folders.deleted_at, folders.id").
		Limit(limit)

	var deleted []models.Folder
	err := tx.Unscoped().Clauses(clause.Returning{}).
		Where("id IN (?) AND deleted_at < ? AND status = ?",
			empty, expiredBefore, models.FolderStatusDeleted).
		Delete(&deleted).Error
	return deleted, err
}
