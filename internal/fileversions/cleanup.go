package fileversions

import (
	"time"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (m Manager) Cleanup(bucketID, fileID, versionID uuid.UUID) error {
	file, err := sql.GetUnscopedFile(m.DB, bucketID, fileID)
	if err != nil || isCurrent(file, versionID) {
		return err
	}

	var version models.FileVersion
	result := m.DB.Where("id = ? AND file_id = ? AND status = ?", versionID, fileID, models.FileStatusDeleting).
		Find(&version)
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}

	if err = m.abortUpload(file, version); err != nil {
		return err
	}
	if err = m.Storage.RemoveObject(storage.VersionObjectKey(bucketID, versionID)); err != nil {
		return err
	}
	if version.CleanupAfter != nil && version.CleanupAfter.After(time.Now()) {
		return nil
	}

	return m.DB.Transaction(func(tx *gorm.DB) error {
		if file, err = sql.LockFile(tx, bucketID, fileID); err != nil || isCurrent(file, versionID) {
			return err
		}
		result = tx.Where("id = ? AND status = ?", versionID, models.FileStatusDeleting).Delete(&models.FileVersion{})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		return deleteFileIfEmpty(tx, file)
	})
}

func (m Manager) ExpirePending(bucketID uuid.UUID, version models.FileVersion, threshold time.Time) (bool, error) {
	if version.UploadID != "" {
		parts, err := m.Storage.ListObjectParts(storage.VersionObjectKey(bucketID, version.ID), version.UploadID)
		if err != nil {
			zap.L().Warn("Failed to list upload parts, expiring version anyway",
				zap.String("version_id", version.ID.String()), zap.Error(err))
		}
		for _, part := range parts {
			if part.LastModified.After(threshold) {
				return false, nil
			}
		}
	}

	expired := false
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := sql.LockFile(tx, bucketID, version.FileID); err != nil {
			return err
		}
		result := tx.Model(&models.FileVersion{}).
			Where("id = ? AND status = ? AND created_at < ?", version.ID, models.FileStatusUploading, threshold).
			Update("status", models.FileStatusDeleting)
		expired = result.RowsAffected > 0
		return result.Error
	})
	return expired, err
}

func deleteFileIfEmpty(tx *gorm.DB, file models.File) error {
	if file.CurrentVersionID != nil || file.Status != models.FileStatusUploading {
		return nil
	}
	var remaining int64
	if err := tx.Model(&models.FileVersion{}).Where("file_id = ?", file.ID).Count(&remaining).Error; err != nil {
		return err
	}
	if remaining > 0 {
		return nil
	}
	return tx.Unscoped().Delete(&file).Error
}
