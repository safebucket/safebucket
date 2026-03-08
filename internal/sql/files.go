package sql

import (
	"errors"
	"fmt"

	"github.com/safebucket/safebucket/internal/database"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetFileByID(db *gorm.DB, bucketID uuid.UUID, fileID uuid.UUID) (models.File, error) {
	var file models.File

	if err := db.Where("id = ? AND bucket_id = ?", fileID, bucketID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.File{}, apierrors.NewAPIError(404, "FILE_NOT_FOUND")
		}
		return models.File{}, err
	}

	return file, nil
}

func GetSharedFilesByDay(db *gorm.DB, days int) []models.TimeSeriesPoint {
	var result []models.TimeSeriesPoint

	startDate := time.Now().AddDate(0, 0, -days)

	dateExpr := database.FormatDateStr(db, "files.created_at")

	// Get files from shared buckets grouped by day
	db.Model(&models.File{}).
		Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateExpr)).
		Where("status = ?", models.FileStatusUploaded).
		Where("files.created_at >= ?", startDate).
		Group(dateExpr).
		Order("date ASC").
		Scan(&result)

	return result
}
