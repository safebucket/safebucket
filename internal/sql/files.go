package sql

import (
	customerrors "api/internal/errors"
	"api/internal/models"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileById(db *gorm.DB, bucketId uuid.UUID, fileId uuid.UUID) (models.File, error) {
	var file models.File

	if err := db.Where("id = ? AND bucket_id = ?", fileId, bucketId).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, customerrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
		return models.File{}, err
	}

	return file, nil
}
