package services

import (
	"net/http"

	"api/internal/helpers"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type HealthService struct {
	DB *gorm.DB
}

func (s HealthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", s.HealthCheck)
	return r
}

func (s HealthService) HealthCheck(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := s.DB.DB()
	if err != nil {
		helpers.RespondWithJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
		return
	}

	if err := sqlDB.PingContext(r.Context()); err != nil {
		helpers.RespondWithJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  "database unreachable",
		})
		return
	}

	helpers.RespondWithJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}
