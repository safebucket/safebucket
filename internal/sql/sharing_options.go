package sql

import (
	"errors"

	apierrors "api/internal/errors"
	"api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetSharingOptionsByFileID retrieves sharing options for a file.
// Returns nil if no sharing options exist for the file.
func GetSharingOptionsByFileID(db *gorm.DB, fileID uuid.UUID) (*models.SharingOptions, error) {
	var sharingOptions models.SharingOptions

	if err := db.Where("file_id = ?", fileID).First(&sharingOptions).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &sharingOptions, nil
}

// IncrementDownloadCount atomically increments the download count for a file's sharing options.
func IncrementDownloadCount(db *gorm.DB, fileID uuid.UUID) error {
	result := db.Model(&models.SharingOptions{}).
		Where("file_id = ?", fileID).
		UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return apierrors.NewAPIError(404, "SHARING_OPTIONS_NOT_FOUND")
	}

	return nil
}
