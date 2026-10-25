package sql

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/safebucket/safebucket/internal/database"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func WithCurrentVersionSize(db *gorm.DB) *gorm.DB {
	return db.Select("files.*, COALESCE(current_file_version.size, 0) AS size").
		Joins("LEFT JOIN file_versions AS current_file_version " +
			"ON current_file_version.id = COALESCE(files.current_version_id, files.id)")
}

func GetFileByID(db *gorm.DB, bucketID uuid.UUID, fileID uuid.UUID) (models.File, error) {
	var file models.File

	if err := db.Where("id = ? AND bucket_id = ?", fileID, bucketID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, apierrors.New(http.StatusNotFound, apierrors.CodeFileNotFound)
		}
		return models.File{}, err
	}

	return file, nil
}

func GetSharedFilesByHour(db *gorm.DB, days int) []models.TimeSeriesPoint {
	var result []models.TimeSeriesPoint

	startDate := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)

	hourExpr := database.FormatHourStr(db, "files.created_at")

	db.Model(&models.File{}).
		Select(fmt.Sprintf("%s as timestamp, COUNT(*) as count", hourExpr)).
		Where("status = ?", models.FileStatusUploaded).
		Where("files.created_at >= ?", startDate).
		Group(hourExpr).
		Order("timestamp ASC").
		Scan(&result)

	return result
}
