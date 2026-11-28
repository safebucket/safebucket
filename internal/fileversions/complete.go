package fileversions

import (
	"errors"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (m Manager) Complete(logger *zap.Logger, bucketID, fileID, versionID uuid.UUID,
	share *models.Share, notification bool,
	onComplete func(tx *gorm.DB, completed Completion) error,
) (Completion, error) {
	file, err := sql.GetUnscopedFile(m.DB, bucketID, fileID)
	if err != nil {
		return Completion{}, err
	}

	version, err := completableVersion(m.DB, file, share, versionID)
	if err != nil || version.Status == models.FileStatusUploaded {
		return Completion{}, err
	}

	// Multipart completion must be validated through the API.
	if notification && version.UploadID != "" {
		return Completion{}, nil
	}

	if err = m.finishStorage(logger, file, version, share); err != nil {
		return Completion{}, err
	}

	var completed Completion
	err = m.DB.Transaction(func(tx *gorm.DB) error {
		lockedFile, lockErr := sql.LockFile(tx, bucketID, fileID)
		if lockErr != nil {
			return lockErr
		}

		pending, versionErr := completableVersion(tx, lockedFile, share, versionID)
		if versionErr != nil || pending.Status == models.FileStatusUploaded {
			return versionErr
		}

		promoted, promoteErr := m.promote(tx, lockedFile, pending)
		if promoteErr != nil {
			return promoteErr
		}

		completed = Completion{File: lockedFile, Version: promoted}
		return onComplete(tx, completed)
	})

	return completed, err
}

func completableVersion(
	db *gorm.DB,
	file models.File,
	share *models.Share,
	versionID uuid.UUID,
) (models.FileVersion, error) {
	if err := checkShareUpload(db, share, file); err != nil {
		return models.FileVersion{}, err
	}
	if err := sql.CheckFileUploadable(file); err != nil {
		return models.FileVersion{}, err
	}
	version, err := sql.GetFileVersionByID(db, file.ID, versionID)
	if err != nil {
		return models.FileVersion{}, err
	}
	if version.Status != models.FileStatusUploaded && version.Status != models.FileStatusUploading {
		return models.FileVersion{}, apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
	}
	return version, nil
}

func (m Manager) promote(tx *gorm.DB, file models.File, version models.FileVersion) (models.FileVersion, error) {
	version.Version = file.LastVersion + 1
	version.Status = models.FileStatusUploaded
	version.UploadID = ""
	if err := tx.Model(&version).Updates(map[string]any{
		"status":    version.Status,
		"version":   version.Version,
		"upload_id": version.UploadID,
	}).Error; err != nil {
		return version, err
	}

	if err := tx.Model(&file).Updates(map[string]any{
		"status":             models.FileStatusUploaded,
		"current_version_id": version.ID,
		"last_version":       version.Version,
	}).Error; err != nil {
		return version, err
	}

	var old []uuid.UUID
	if err := tx.Model(&models.FileVersion{}).
		Where("file_id = ? AND status = ? AND id != ?", file.ID, models.FileStatusUploaded, version.ID).
		Order("version DESC").
		Offset(m.MaxVersions-1).
		Pluck("id", &old).
		Error; err != nil {
		return version, err
	}
	if len(old) == 0 {
		return version, nil
	}

	return version, tx.Model(&models.FileVersion{}).Where("id IN ?", old).
		Update("status", models.FileStatusDeleting).Error
}

func (m Manager) finishStorage(
	logger *zap.Logger,
	file models.File,
	version models.FileVersion,
	share *models.Share,
) error {
	key := storage.VersionObjectKey(file.BucketID, version.ID)
	if _, err := m.Storage.StatObject(key); err == nil {
		return nil
	}

	if version.UploadID == "" {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotInStorage)
	}

	err := storage.FinalizeMultipartUpload(
		m.Storage,
		key,
		version.UploadID,
		int64(version.Size),
		Metadata(file, version, share),
	)
	if err != nil {
		if _, statErr := m.Storage.StatObject(key); statErr == nil {
			return nil
		}
	}

	if errors.Is(err, storage.ErrMultipartPartMismatch) {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeMultipartSizeMismatch)
	}
	if err != nil {
		logger.Error(
			"Failed to complete multipart upload",
			zap.Error(err),
			zap.String("version_id", version.ID.String()),
		)
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeMultipartCompleteFailed)
	}
	return nil
}
