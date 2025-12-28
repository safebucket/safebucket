package services

import (
	m "api/internal/middlewares"
	"api/internal/models"

	"api/internal/handlers"

	"api/internal/sql"

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

	response.SharedFiles = sql.GetSharedFilesByDay(s.DB, queryParams.Days)

	return response, nil
}
