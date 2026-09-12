package sql

import (
	"errors"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetBucketByID(db *gorm.DB, bucketID uuid.UUID) (models.Bucket, error) {
	var bucket models.Bucket

	if err := db.Where("id = ?", bucketID).First(&bucket).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Bucket{}, apierrors.New(http.StatusNotFound, apierrors.CodeBucketNotFound)
		}
		return models.Bucket{}, err
	}

	return bucket, nil
}
