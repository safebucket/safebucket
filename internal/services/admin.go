package services

import (
	"time"

	m "api/internal/middlewares"
	"api/internal/models"

	"api/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminService struct {
	DB *gorm.DB
}

func (s AdminService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		With(m.ValidateQuery[models.AdminStatsQueryParams]).
		Get("/stats", handlers.GetOneWithQueryHandler(s.GetStats))

	return r
}

func (s AdminService) GetStats(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	queryParams models.AdminStatsQueryParams,
) (models.AdminStatsResponse, error) {
	var response models.AdminStatsResponse

	s.DB.Model(&models.User{}).Count(&response.TotalUsers)

	s.DB.Model(&models.Bucket{}).Count(&response.TotalBuckets)

	s.DB.Model(&models.File{}).
		Where("status = ?", models.FileStatusUploaded).
		Count(&response.TotalFiles)

	s.DB.Model(&models.Folder{}).Count(&response.TotalFolders)

	var totalStorage *int64
	s.DB.Model(&models.File{}).
		Where("status = ?", models.FileStatusUploaded).
		Select("COALESCE(SUM(size), 0)").
		Scan(&totalStorage)
	if totalStorage != nil {
		response.TotalStorageBytes = *totalStorage
	}

	var roleDistribution []models.RoleCount
	s.DB.Model(&models.User{}).
		Select("role, COUNT(*) as count").
		Group("role").
		Scan(&roleDistribution)
	response.RoleDistribution = roleDistribution

	var providerDistribution []models.ProviderCount
	s.DB.Model(&models.User{}).
		Select("provider_type as provider, COUNT(*) as count").
		Group("provider_type").
		Scan(&providerDistribution)
	response.ProviderDistribution = providerDistribution

	response.SharedFiles = s.getSharedFilesByDay(queryParams.Days)

	return response, nil
}

func (s AdminService) getSharedFilesByDay(days int) []models.TimeSeriesPoint {
	var result []models.TimeSeriesPoint

	startDate := time.Now().AddDate(0, 0, -days)

	// Get files from shared buckets grouped by day
	s.DB.Model(&models.File{}).
		Select("TO_CHAR(files.created_at, 'YYYY-MM-DD') as date, COUNT(*) as count").
		Where("status = ?", models.FileStatusUploaded).
		Where("files.created_at >= ?", startDate).
		Group("TO_CHAR(files.created_at, 'YYYY-MM-DD')").
		Order("date ASC").
		Scan(&result)

	return result
}
