package fileversions

import (
	"net/http"
	"slices"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (m Manager) Delete(
	logger *zap.Logger,
	bucketID, fileID, versionID uuid.UUID,
	share *models.Share,
	onDelete func(tx *gorm.DB, file models.File, version models.FileVersion) error,
) error {
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		file, version, changed, err := markDeleting(tx, bucketID, fileID, versionID, share)
		if err != nil || !changed || onDelete == nil {
			return err
		}
		return onDelete(tx, file, version)
	})
	if err != nil {
		return err
	}

	if err = m.Cleanup(bucketID, fileID, versionID); err != nil {
		logger.Warn("Version deletion will be retried", zap.Error(err), zap.String("version_id", versionID.String()))
	}
	return nil
}

func markDeleting(
	tx *gorm.DB,
	bucketID, fileID, versionID uuid.UUID,
	share *models.Share,
) (models.File, models.FileVersion, bool, error) {
	file, err := sql.LockFile(tx, bucketID, fileID)
	if err != nil {
		return models.File{}, models.FileVersion{}, false, err
	}
	version, err := sql.GetFileVersionByID(tx, fileID, versionID)
	if err != nil {
		return models.File{}, models.FileVersion{}, false, err
	}
	if err = checkShareUpload(tx, share, file); err != nil {
		return models.File{}, models.FileVersion{}, false, err
	}

	allowed := []models.FileStatus{models.FileStatusUploading, models.FileStatusDeleting}
	if share == nil {
		allowed = append(allowed, models.FileStatusUploaded)
	}
	if !slices.Contains(allowed, version.Status) {
		return models.File{}, models.FileVersion{}, false,
			apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotDeletable)
	}
	if isCurrent(file, versionID) {
		return models.File{}, models.FileVersion{}, false,
			apierrors.New(http.StatusConflict, apierrors.CodeFileVersionIsCurrent)
	}
	if version.Status == models.FileStatusDeleting {
		return file, version, false, nil
	}

	updates := map[string]any{"status": models.FileStatusDeleting}
	if version.Status == models.FileStatusUploading {
		// Keep a tombstone while an already issued upload URL can still be used.
		updates["cleanup_after"] = time.Now().Add(configuration.UploadPolicyExpirationInMinutes * time.Minute)
	}
	err = tx.Model(&models.FileVersion{ID: version.ID}).Updates(updates).Error
	return file, version, err == nil, err
}
