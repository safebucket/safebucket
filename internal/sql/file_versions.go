package sql

import (
	"net/http"
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/helpers"
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

func GetUploadVersion(tx *gorm.DB, file models.File) (models.FileVersion, error) {
	var version models.FileVersion
	result := tx.Where("file_id = ? AND status = ?", file.ID, models.FileStatusUploading).Find(&version)
	if result.Error != nil {
		return version, result.Error
	}
	if result.RowsAffected != 0 {
		return version, nil
	}
	if file.CurrentVersionID != nil {
		return GetFileVersionByID(tx, file.ID, *file.CurrentVersionID)
	}
	return version, apierrors.New(http.StatusConflict, apierrors.CodeInvalidFileStatusTransition)
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
	var file models.File
	err := tx.Unscoped().Where("id = ?", fileID).Find(&file).Error
	return file, err
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

func ReserveFileVersion(tx *gorm.DB, bucketID uuid.UUID, body models.FileUploadBody,
	userID *uuid.UUID, share *models.Share,
) (models.File, models.FileVersion, error) {
	if err := LockBucketFileNames(tx, bucketID); err != nil {
		return models.File{}, models.FileVersion{}, err
	}
	if err := validateUploadFolder(tx, bucketID, body.FolderID); err != nil {
		return models.File{}, models.FileVersion{}, err
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
		return file, models.FileVersion{}, result.Error
	}
	newFile := result.RowsAffected == 0
	if newFile {
		file = models.File{ID: uuid.New(), BucketID: bucketID, FolderID: body.FolderID, Name: body.Name,
			Extension: helpers.ExtensionFromName(body.Name), Size: body.Size, Status: models.FileStatusUploading,
			ExpiresAt: body.ExpiresAt}
		if err := tx.Create(&file).Error; err != nil {
			return file, models.FileVersion{}, err
		}
		if share != nil && share.Type == models.ShareTypeFiles {
			if err := tx.Create(&models.ShareFile{ShareID: share.ID, FileID: file.ID}).Error; err != nil {
				return file, models.FileVersion{}, err
			}
		}
	} else {
		var err error
		file, err = LockFile(tx, bucketID, file.ID)
		if err != nil {
			return file, models.FileVersion{}, err
		}
		if err = CheckFileUploadable(file); err != nil {
			return file, models.FileVersion{}, err
		}
	}
	if share != nil && !helpers.IsFileInShare(tx, *share, file.ID, file) {
		return file, models.FileVersion{}, apierrors.New(http.StatusForbidden, apierrors.CodeShareFileNotInShare)
	}
	var pending int64
	if err := tx.Model(&models.FileVersion{}).Where("file_id = ? AND status = ?", file.ID, models.FileStatusUploading).
		Count(&pending).Error; err != nil {
		return file, models.FileVersion{}, err
	}
	if pending > 0 {
		return file, models.FileVersion{}, apierrors.New(http.StatusConflict, apierrors.CodeFileUploadInProgress)
	}
	var nextVersion int
	if err := tx.Model(&models.FileVersion{}).Where("file_id = ?", file.ID).
		Select("COALESCE(MAX(version), 0) + 1").Scan(&nextVersion).Error; err != nil {
		return file, models.FileVersion{}, err
	}
	version := models.FileVersion{ID: uuid.New(), FileID: file.ID, Version: nextVersion,
		Size: body.Size, Status: models.FileStatusUploading, UploadedBy: userID}
	if newFile {
		version.ID = file.ID
	}
	if share != nil {
		version.ShareID = &share.ID
		result = tx.Model(&models.Share{}).
			Where("id = ? AND allow_upload = ? AND (max_uploads IS NULL OR current_uploads < max_uploads)", share.ID, true).
			UpdateColumn("current_uploads", gorm.Expr("current_uploads + 1"))
		if result.Error != nil {
			return file, version, result.Error
		}
		if result.RowsAffected == 0 {
			return file, version, apierrors.New(http.StatusForbidden, apierrors.CodeShareMaxUploadsReached)
		}
	}
	err := tx.Create(&version).Error
	return file, version, err
}

func PromoteFileVersion(tx *gorm.DB, file models.File, version models.FileVersion, limit int) error {
	if err := tx.Model(&version).Update("status", models.FileStatusUploaded).Error; err != nil {
		return err
	}
	if err := tx.Model(&file).Updates(map[string]interface{}{
		"status": models.FileStatusUploaded, "size": version.Size, "current_version_id": version.ID,
	}).Error; err != nil {
		return err
	}
	var old []uuid.UUID
	if err := tx.Model(&models.FileVersion{}).
		Where("file_id = ? AND status = ? AND id != ?", file.ID, models.FileStatusUploaded, version.ID).
		Order("version DESC").
		Offset(limit-1).
		Pluck("id", &old).
		Error; err != nil {
		return err
	}
	if len(old) == 0 {
		return nil
	}
	return tx.Model(&models.FileVersion{}).Where("id IN ?", old).Update("status", models.FileStatusDeleting).Error
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
