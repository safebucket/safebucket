package sql

import (
	"net/http"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileVersionByID(db *gorm.DB, fileID, versionID uuid.UUID) (models.FileVersion, error) {
	var version models.FileVersion
	result := db.Where("id = ? AND file_id = ?", versionID, fileID).Find(&version)
	if result.Error != nil {
		return models.FileVersion{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.FileVersion{}, apierrors.New(http.StatusNotFound, apierrors.CodeFileVersionNotFound)
	}

	return version, nil
}

func LockBucketFileNames(tx *gorm.DB, bucketID uuid.UUID) error {
	result := tx.Model(&models.Bucket{}).Where("id = ?", bucketID).UpdateColumn("name", gorm.Expr("name"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
	}
	return nil
}

func LockFile(tx *gorm.DB, bucketID, fileID uuid.UUID) (models.File, error) {
	result := tx.Unscoped().Model(&models.File{}).Where("id = ? AND bucket_id = ?", fileID, bucketID).
		UpdateColumn("name", gorm.Expr("name"))
	if result.Error != nil {
		return models.File{}, result.Error
	}

	if result.RowsAffected == 0 {
		return models.File{}, apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
	}
	return GetUnscopedFile(tx, bucketID, fileID)
}

func GetUnscopedFile(db *gorm.DB, bucketID, fileID uuid.UUID) (models.File, error) {
	var file models.File
	result := db.Unscoped().Where("id = ? AND bucket_id = ?", fileID, bucketID).Find(&file)
	if result.Error != nil {
		return models.File{}, result.Error
	}

	if result.RowsAffected == 0 {
		return models.File{}, apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
	}
	return file, nil
}

func CheckFileUploadable(file models.File) error {
	if file.DeletedAt.Valid || (file.Status != models.FileStatusUploading && file.Status != models.FileStatusUploaded) {
		return apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
	}
	if file.ExpiresAt != nil && !file.ExpiresAt.After(time.Now()) {
		return apierrors.New(http.StatusForbidden, apierrors.CodeFileExpired)
	}
	return nil
}
