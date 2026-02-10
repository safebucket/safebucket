package services

import (
	"net/http"

	"api/internal/activity"
	"api/internal/cache"
	apierrors "api/internal/errors"
	"api/internal/handlers"
	"api/internal/models"
	"api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type HealthService struct {
	DB             *gorm.DB
	Cache          cache.ICache
	ActivityLogger activity.IActivityLogger
	Storage        storage.IStorage
}

func (s HealthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", handlers.GetOneHandler(s.HealthCheck))
	return r
}

func (s HealthService) HealthCheck(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs) (models.HealthResponse, error) {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return models.HealthResponse{}, apierrors.NewAPIError(http.StatusServiceUnavailable, "DATABASE_UNREACHABLE")
	}

	if err = sqlDB.Ping(); err != nil {
		return models.HealthResponse{}, apierrors.NewAPIError(http.StatusServiceUnavailable, "DATABASE_UNREACHABLE")
	}

	if err = s.Cache.Ping(); err != nil {
		return models.HealthResponse{}, apierrors.NewAPIError(http.StatusServiceUnavailable, "CACHE_UNREACHABLE")
	}

	if err = s.ActivityLogger.Ping(); err != nil {
		return models.HealthResponse{}, apierrors.NewAPIError(http.StatusServiceUnavailable, "ACTIVITY_LOGGER_UNREACHABLE")
	}

	if err = s.Storage.Ping(); err != nil {
		return models.HealthResponse{}, apierrors.NewAPIError(http.StatusServiceUnavailable, "STORAGE_UNREACHABLE")
	}

	return models.HealthResponse{Status: "ok"}, nil
}
