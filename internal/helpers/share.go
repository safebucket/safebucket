package helpers

import (
	"time"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetShareFile(db *gorm.DB, share models.Share, fileID uuid.UUID) (models.File, error) {
	var file models.File
	now := time.Now()

	query := db.
		Where("files.id = ?", fileID).
		Where("files.bucket_id = ?", share.BucketID).
		Where("files.status = ?", models.FileStatusUploaded).
		Where("files.expires_at IS NULL OR files.expires_at > ?", now)

	switch share.Type {
	case models.ShareTypeFiles:
		query = query.Joins("JOIN share_files ON share_files.file_id = files.id").
			Where("share_files.share_id = ?", share.ID)
	case models.ShareTypeFolder, models.ShareTypeBucket:
	default:
		return models.File{}, apierrors.NewAPIError(404, "FILE_NOT_FOUND")
	}

	if err := query.First(&file).Error; err != nil {
		return models.File{}, apierrors.NewAPIError(404, "FILE_NOT_FOUND")
	}

	if share.Type == models.ShareTypeFolder {
		if share.FolderID == nil || !IsFolderDescendant(db, file.FolderID, *share.FolderID) {
			return models.File{}, apierrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
	}

	return file, nil
}

func IsFileInShare(db *gorm.DB, share models.Share, fileID uuid.UUID, file models.File) bool {
	switch share.Type {
	case models.ShareTypeFiles:
		var count int64
		db.Model(&models.ShareFile{}).
			Where("share_id = ? AND file_id = ?", share.ID, fileID).
			Count(&count)
		return count > 0

	case models.ShareTypeFolder:
		if share.FolderID == nil {
			return false
		}
		return IsFolderDescendant(db, file.FolderID, *share.FolderID)

	case models.ShareTypeBucket:
		return file.BucketID == share.BucketID

	default:
		return false
	}
}

func IsFolderDescendant(db *gorm.DB, folderID *uuid.UUID, targetFolderID uuid.UUID) bool {
	if folderID == nil {
		return false
	}
	currentID := folderID
	for i := 0; i < 100 && currentID != nil; i++ {
		if *currentID == targetFolderID {
			return true
		}
		var folder models.Folder
		if err := db.Where("id = ?", currentID).First(&folder).Error; err != nil {
			return false
		}
		currentID = folder.FolderID
	}
	return false
}
