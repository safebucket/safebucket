package sql

import (
	"errors"

	customerrors "api/internal/errors"
	"api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileByID(db *gorm.DB, bucketID uuid.UUID, fileID uuid.UUID) (models.File, error) {
	var file models.File

	if err := db.Where("id = ? AND bucket_id = ?", fileID, bucketID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, customerrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
		return models.File{}, err
	}

	return file, nil
}
