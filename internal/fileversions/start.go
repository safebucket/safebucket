package fileversions

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (m Manager) Start(logger *zap.Logger, bucketID uuid.UUID, body models.FileUploadBody,
	userID *uuid.UUID, share *models.Share,
) (models.FileUploadResponse, error) {
	var file models.File
	var version models.FileVersion
	err := m.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		file, version, err = reserve(tx, bucketID, body, userID, share)
		return err
	})
	if err != nil {
		return models.FileUploadResponse{}, err
	}

	key := storage.VersionObjectKey(bucketID, version.ID)
	presigned, err := m.Storage.PresignUpload(key, body.Size, Metadata(file, version, share))
	if err == nil && presigned.UploadID != "" {
		if err = m.DB.Model(&version).Update("upload_id", presigned.UploadID).Error; err != nil {
			if abortErr := m.Storage.AbortMultipartUpload(key, presigned.UploadID); abortErr != nil {
				logger.Warn("Failed to abort uncommitted upload", zap.Error(abortErr))
			}
		}
	}
	if err != nil {
		releaseErr := m.DB.Transaction(func(tx *gorm.DB) error {
			return release(tx, bucketID, version, share)
		})
		if releaseErr != nil {
			logger.Warn("Failed to release upload reservation", zap.Error(releaseErr))
		}
		return models.FileUploadResponse{}, err
	}

	response := presigned.Response
	response.ID = file.ID.String()
	response.VersionID = version.ID.String()
	return response, nil
}

func reserve(tx *gorm.DB, bucketID uuid.UUID, body models.FileUploadBody,
	userID *uuid.UUID, share *models.Share,
) (models.File, models.FileVersion, error) {
	if err := sql.LockBucketFileNames(tx, bucketID); err != nil {
		return models.File{}, models.FileVersion{}, err
	}
	file, created, err := findOrCreateFile(tx, bucketID, body, share)
	if err != nil {
		return models.File{}, models.FileVersion{}, err
	}
	if share != nil && !helpers.IsFileInShare(tx, *share, file.ID, file) {
		return file, models.FileVersion{}, apierrors.New(http.StatusForbidden, apierrors.CodeShareFileNotInShare)
	}

	var pending int64
	if err = tx.Model(&models.FileVersion{}).Where("file_id = ? AND status = ?", file.ID, models.FileStatusUploading).
		Count(&pending).Error; err != nil {
		return file, models.FileVersion{}, err
	}
	if pending > 0 {
		return file, models.FileVersion{}, apierrors.New(http.StatusConflict, apierrors.CodeFileUploadInProgress)
	}

	version := models.FileVersion{ID: uuid.New(), FileID: file.ID,
		Size: body.Size, Status: models.FileStatusUploading, UploadedBy: userID}
	if created {
		version.ID = file.ID
	}
	if share != nil {
		result := tx.Model(&models.Share{}).
			Where("id = ? AND allow_upload = ? AND (max_uploads IS NULL OR current_uploads < max_uploads)", share.ID, true).
			UpdateColumn("current_uploads", gorm.Expr("current_uploads + 1"))
		if result.Error != nil {
			return file, version, result.Error
		}
		if result.RowsAffected == 0 {
			return file, version, apierrors.New(http.StatusForbidden, apierrors.CodeShareMaxUploadsReached)
		}
	}
	return file, version, tx.Create(&version).Error
}

func findOrCreateFile(
	tx *gorm.DB,
	bucketID uuid.UUID,
	body models.FileUploadBody,
	share *models.Share,
) (models.File, bool, error) {
	if err := validateUploadFolder(tx, bucketID, body.FolderID); err != nil {
		return models.File{}, false, err
	}

	var file models.File
	query := tx.Where("bucket_id = ? AND name = ?", bucketID, body.Name)
	if body.FolderID == nil {
		query = query.Where("folder_id IS NULL")
	} else {
		query = query.Where("folder_id = ?", body.FolderID)
	}
	result := query.Find(&file)
	if result.Error != nil {
		return models.File{}, false, result.Error
	}
	if result.RowsAffected > 0 {
		locked, err := sql.LockFile(tx, bucketID, file.ID)
		if err != nil {
			return models.File{}, false, err
		}
		return locked, false, sql.CheckFileUploadable(locked)
	}

	file = models.File{ID: uuid.New(), BucketID: bucketID, FolderID: body.FolderID, Name: body.Name,
		Extension: helpers.ExtensionFromName(body.Name), Status: models.FileStatusUploading,
		ExpiresAt: body.ExpiresAt}
	if err := tx.Create(&file).Error; err != nil {
		return models.File{}, false, err
	}
	if share != nil && share.Type == models.ShareTypeFiles {
		if err := tx.Create(&models.ShareFile{ShareID: share.ID, FileID: file.ID}).Error; err != nil {
			return models.File{}, false, err
		}
	}
	return file, true, nil
}

func release(tx *gorm.DB, bucketID uuid.UUID, version models.FileVersion, share *models.Share) error {
	file, err := sql.LockFile(tx, bucketID, version.FileID)
	if err != nil {
		return err
	}

	result := tx.Where("id = ? AND status = ?", version.ID, models.FileStatusUploading).Delete(&models.FileVersion{})
	if result.Error != nil || result.RowsAffected == 0 {
		return result.Error
	}

	if share != nil {
		if err = tx.Model(&models.Share{}).Where("id = ? AND current_uploads > 0", share.ID).
			UpdateColumn("current_uploads", gorm.Expr("current_uploads - 1")).Error; err != nil {
			return err
		}
	}
	return deleteFileIfEmpty(tx, file)
}

func validateUploadFolder(tx *gorm.DB, bucketID uuid.UUID, folderID *uuid.UUID) error {
	if folderID == nil {
		return nil
	}
	var folder models.Folder
	result := tx.Where("id = ? AND bucket_id = ? AND status = ?", folderID, bucketID, models.FolderStatusCreated).
		Find(&folder)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeFolderNotFound)
	}
	return nil
}
