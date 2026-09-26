package fileversions

import (
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (m Manager) Cleanup(bucketID, fileID, versionID uuid.UUID) error {
	return m.DB.Transaction(func(tx *gorm.DB) error {
		file, err := sql.LockFile(tx, bucketID, fileID)
		if err != nil {
			return err
		}
		var version models.FileVersion
		result := tx.Where("id = ? AND file_id = ? AND status = ?", versionID, fileID, models.FileStatusDeleting).
			Find(&version)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if file.CurrentVersionID != nil && *file.CurrentVersionID == versionID {
			return nil
		}
		if err = m.abortUpload(file, version); err != nil {
			return err
		}
		if err = m.Storage.RemoveObject(storage.VersionObjectKey(file.BucketID, version.ID)); err != nil {
			return err
		}
		if version.CleanupAfter != nil && version.CleanupAfter.After(time.Now()) {
			return nil
		}
		if err = tx.Delete(&version).Error; err != nil {
			return err
		}
		if file.CurrentVersionID == nil && file.Status == models.FileStatusUploading {
			var remaining int64
			if err = tx.Model(&models.FileVersion{}).Where("file_id = ?", fileID).Count(&remaining).Error; err != nil {
				return err
			}
			if remaining == 0 {
				return tx.Unscoped().Delete(&file).Error
			}
		}
		return nil
	})
}

// ExpirePending marks a stale upload for deletion; the deleted versions cleanup removes it.
func (m Manager) ExpirePending(bucketID uuid.UUID, version models.FileVersion, threshold time.Time) (bool, error) {
	expired := false
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := sql.LockFile(tx, bucketID, version.FileID); err != nil {
			return err
		}
		result := tx.Where("id = ? AND status = ? AND created_at < ?", version.ID, models.FileStatusUploading, threshold).
			Find(&version)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		multipart, isMultipart, err := cache.GetMultipartState(m.Cache, version.ID.String())
		if err != nil {
			return err
		}
		if isMultipart {
			key := storage.VersionObjectKey(bucketID, version.ID)
			parts, listErr := m.Storage.ListObjectParts(key, multipart.UploadID)
			if listErr != nil {
				return listErr
			}
			for _, part := range parts {
				if part.LastModified.After(threshold) {
					return nil
				}
			}
		}
		if err = releaseUploadQuota(tx, version); err != nil {
			return err
		}
		if err = tx.Model(&version).Update("status", models.FileStatusDeleting).Error; err != nil {
			return err
		}
		expired = true
		return nil
	})
	return expired, err
}

func (m Manager) RemoveFileObjects(tx *gorm.DB, files ...models.File) error {
	var keys []string
	for _, file := range files {
		versions, err := m.prepareRemoval(tx, file)
		if err != nil {
			return err
		}
		for _, version := range versions {
			keys = append(keys, storage.VersionObjectKey(file.BucketID, version.ID))
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return m.Storage.RemoveObjects(keys)
}

func (m Manager) prepareRemoval(tx *gorm.DB, file models.File) ([]models.FileVersion, error) {
	if _, err := sql.LockFile(tx, file.BucketID, file.ID); err != nil {
		return nil, err
	}
	var versions []models.FileVersion
	if err := tx.Where("file_id = ?", file.ID).Find(&versions).Error; err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		versions = append(versions, models.FileVersion{ID: file.ID, Version: 1})
	}
	for _, version := range versions {
		if err := m.abortUpload(file, version); err != nil {
			return nil, err
		}
		if err := releaseUploadQuota(tx, version); err != nil {
			return nil, err
		}
	}
	return versions, nil
}

func (m Manager) abortUpload(file models.File, version models.FileVersion) error {
	multipart, isMultipart, err := cache.GetMultipartState(m.Cache, version.ID.String())
	if err != nil || !isMultipart {
		return err
	}
	key := storage.VersionObjectKey(file.BucketID, version.ID)
	if err = m.Storage.AbortMultipartUpload(key, multipart.UploadID); err != nil {
		return err
	}
	return cache.DeleteMultipartState(m.Cache, version.ID.String())
}

func releaseUploadQuota(tx *gorm.DB, version models.FileVersion) error {
	if version.ShareID == nil || version.Status != models.FileStatusUploading {
		return nil
	}
	return tx.Model(&models.Share{}).
		Where("id = ? AND current_uploads > 0", *version.ShareID).
		UpdateColumn("current_uploads", gorm.Expr("current_uploads - 1")).Error
}
