package fileversions

import (
	"errors"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Manager struct {
	DB          *gorm.DB
	Storage     storage.IStorage
	Cache       cache.ICache
	MaxVersions int
}

type Completion struct {
	File    models.File
	Version models.FileVersion
	Changed bool
}

func (m Manager) Start(logger *zap.Logger, bucketID uuid.UUID, body models.FileUploadBody,
	userID *uuid.UUID, share *models.Share,
) (models.FileUploadResponse, error) {
	var file models.File
	var version models.FileVersion
	var presigned storage.PresignedUpload
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		file, version, err = sql.ReserveFileVersion(tx, bucketID, body, userID, share)
		if err != nil {
			return err
		}
		presigned, err = m.Storage.PresignUpload(
			storage.VersionObjectKey(bucketID, version.ID),
			body.Size,
			Metadata(file, version),
		)
		if err != nil || presigned.UploadID == "" {
			return err
		}
		return cache.SetMultipartState(m.Cache, version.ID.String(), cache.MultipartState{
			UploadID: presigned.UploadID, PartSize: presigned.PartSize,
		})
	})
	if err != nil {
		if presigned.UploadID != "" {
			if abortErr := m.Storage.AbortMultipartUpload(
				storage.VersionObjectKey(bucketID, version.ID),
				presigned.UploadID,
			); abortErr != nil {
				logger.Warn("Failed to abort uncommitted upload", zap.Error(abortErr))
			}
		}
		return models.FileUploadResponse{}, err
	}
	response := presigned.Response
	response.ID = file.ID.String()
	response.VersionID = version.ID.String()
	return response, nil
}

func (m Manager) Complete(logger *zap.Logger, bucketID, fileID, versionID uuid.UUID,
	share *models.Share, notification bool,
) (Completion, error) {
	var completed Completion
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		file, err := sql.LockFile(tx, bucketID, fileID)
		if err != nil {
			return err
		}
		if err = sql.CheckFileUploadable(file); err != nil {
			return err
		}
		var version models.FileVersion
		if versionID == uuid.Nil {
			version, err = sql.GetUploadVersion(tx, file)
		} else {
			version, err = sql.GetFileVersionByID(tx, fileID, versionID)
		}
		if err != nil {
			return err
		}
		if err = authorizeShare(tx, share, file, version); err != nil {
			return err
		}
		if version.Status == models.FileStatusUploaded {
			return nil
		}
		if version.Status != models.FileStatusUploading {
			return apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
		}
		multipart, isMultipart, err := cache.GetMultipartState(m.Cache, version.ID.String())
		if err != nil {
			return err
		}
		// Multipart completion must be validated through the API.
		if notification && isMultipart {
			return nil
		}
		if err = m.finishStorage(logger, file, version, multipart); err != nil {
			return err
		}
		if err = sql.PromoteFileVersion(tx, file, version, m.MaxVersions); err != nil {
			return err
		}
		completed = Completion{File: file, Version: version, Changed: true}
		return nil
	})
	if err == nil && completed.Changed {
		if cacheErr := cache.DeleteMultipartState(m.Cache, completed.Version.ID.String()); cacheErr != nil {
			logger.Warn("Failed to remove multipart state", zap.Error(cacheErr))
		}
	}
	return completed, err
}

// Delete returns the file and version as they were before the deletion.
func (m Manager) Delete(
	logger *zap.Logger,
	bucketID, fileID, versionID uuid.UUID,
	share *models.Share,
) (models.File, models.FileVersion, error) {
	var file models.File
	var version models.FileVersion
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		file, err = sql.LockFile(tx, bucketID, fileID)
		if err != nil {
			return err
		}
		version, err = sql.GetFileVersionByID(tx, fileID, versionID)
		if err != nil {
			return err
		}
		if err = authorizeShare(tx, share, file, version); err != nil {
			return err
		}
		if share != nil && version.Status != models.FileStatusUploading && version.Status != models.FileStatusDeleting {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotDeletable)
		}
		if file.CurrentVersionID != nil && *file.CurrentVersionID == versionID {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionIsCurrent)
		}
		if version.Status == models.FileStatusDeleting {
			return nil
		}
		if version.Status != models.FileStatusUploading && version.Status != models.FileStatusUploaded {
			return apierrors.New(http.StatusConflict, apierrors.CodeFileVersionNotDeletable)
		}
		updates := map[string]interface{}{"status": models.FileStatusDeleting}
		if version.Status == models.FileStatusUploading {
			if err = releaseUploadQuota(tx, version); err != nil {
				return err
			}
			// Keep a tombstone while an already issued upload URL can still be used.
			updates["cleanup_after"] = time.Now().Add(configuration.UploadPolicyExpirationInMinutes * time.Minute)
		}
		return tx.Model(&models.FileVersion{ID: version.ID}).Updates(updates).Error
	})
	if err != nil {
		return models.File{}, models.FileVersion{}, err
	}
	if err = m.Cleanup(bucketID, fileID, versionID); err != nil {
		logger.Warn("Version deletion will be retried", zap.Error(err), zap.String("version_id", versionID.String()))
	}
	return file, version, nil
}

func Metadata(file models.File, version models.FileVersion) map[string]string {
	metadata := map[string]string{
		"bucket_id":  file.BucketID.String(),
		"file_id":    file.ID.String(),
		"version_id": version.ID.String(),
	}
	if version.UploadedBy != nil {
		metadata["user_id"] = version.UploadedBy.String()
	}
	if version.ShareID != nil {
		metadata["share_id"] = version.ShareID.String()
	}
	return metadata
}

func (m Manager) finishStorage(
	logger *zap.Logger,
	file models.File,
	version models.FileVersion,
	multipart cache.MultipartState,
) error {
	key := storage.VersionObjectKey(file.BucketID, version.ID)
	if _, err := m.Storage.StatObject(key); err == nil {
		return nil
	}
	if multipart.UploadID == "" {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFileNotInStorage)
	}
	err := storage.FinalizeMultipartUpload(
		m.Storage,
		key,
		multipart.UploadID,
		multipart.PartSize,
		int64(version.Size),
		Metadata(file, version),
	)
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

func authorizeShare(tx *gorm.DB, share *models.Share, file models.File, version models.FileVersion) error {
	if share == nil {
		return nil
	}
	if !share.AllowUpload {
		return apierrors.New(http.StatusForbidden, apierrors.CodeShareUploadNotAllowed)
	}
	if !helpers.IsFileInShare(tx, *share, file.ID, file) {
		return apierrors.New(http.StatusForbidden, apierrors.CodeShareFileNotInShare)
	}
	if version.ShareID == nil || *version.ShareID != share.ID {
		return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}
	return nil
}
