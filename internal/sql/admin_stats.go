package sql

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func storageBreakdownScope(db *gorm.DB) *gorm.DB {
	return db.Model(&models.File{}).
		Joins("LEFT JOIN file_versions ON file_versions.file_id = files.id AND file_versions.status = ?",
			models.FileStatusUploaded).
		Where("files.deleted_at IS NULL").
		Where("files.status = ?", models.FileStatusUploaded)
}

func scanStorageBreakdown(query *gorm.DB) (models.StorageBreakdown, error) {
	var breakdown models.StorageBreakdown
	if err := query.Select(`COALESCE(SUM(COALESCE(file_versions.size, files.size)), 0) AS total,
COALESCE(SUM(CASE WHEN file_versions.id = files.current_version_id THEN file_versions.size WHEN file_versions.id IS NULL THEN files.size ELSE 0 END), 0) AS active`).
		Scan(&breakdown).Error; err != nil {
		return models.StorageBreakdown{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	breakdown.Inactive = breakdown.Total - breakdown.Active

	return breakdown, nil
}

func SumStorageBytes(db *gorm.DB) (models.StorageBreakdown, error) {
	return scanStorageBreakdown(storageBreakdownScope(db))
}

func SumBucketStorageBytes(db *gorm.DB, bucketID uuid.UUID) (models.StorageBreakdown, error) {
	return scanStorageBreakdown(
		storageBreakdownScope(db).Where("files.bucket_id = ?", bucketID),
	)
}
