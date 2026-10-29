package sql

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func storageScope(db *gorm.DB) *gorm.DB {
	return db.Model(&models.File{}).
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id AND file_versions.status = ?",
			models.FileStatusUploaded).
		Where("files.deleted_at IS NULL").
		Where("files.status = ?", models.FileStatusUploaded)
}

func scanStorage(query *gorm.DB) (int64, error) {
	var total int64
	if err := query.Select("COALESCE(SUM(file_versions.size), 0)").Scan(&total).Error; err != nil {
		return 0, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	return total, nil
}

func SumStorageBytes(db *gorm.DB) (int64, error) {
	return scanStorage(storageScope(db))
}

func SumBucketStorageBytes(db *gorm.DB, bucketID uuid.UUID) (int64, error) {
	return scanStorage(storageScope(db).Where("files.bucket_id = ?", bucketID))
}
