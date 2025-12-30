package services

import (
	"api/internal/activity"
	"api/internal/handlers"
	m "api/internal/middlewares"
	"api/internal/models"

	"api/internal/sql"

	"api/internal/rbac"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminService struct {
	DB             *gorm.DB
	ActivityLogger activity.IActivityLogger
}

func (s AdminService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		With(m.ValidateQuery[models.AdminStatsQueryParams]).
		Get("/stats", handlers.GetOneWithQueryHandler(s.GetStats))

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		Get("/activity", handlers.GetListHandler(s.GetActivity))

	r.With(m.AuthorizeRole(models.RoleAdmin)).
		Get("/buckets", handlers.GetListHandler(s.GetBucketList))

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

	searchCriteria := map[string][]string{
		"action":      {rbac.ActionCreate.String()},
		"object_type": {rbac.ResourceFile.String()},
	}

	timeSeries, err := s.ActivityLogger.CountByDay(searchCriteria, queryParams.Days)
	if err != nil {
		zap.L().Error("Failed to get uploads per day from Loki, falling back to DB", zap.Error(err))
		response.SharedFilesPerDay = sql.GetSharedFilesByDay(s.DB, queryParams.Days)
	} else {
		response.SharedFilesPerDay = timeSeries
	}

	return response, nil
}

func (s AdminService) GetActivity(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
) []map[string]interface{} {
	searchCriteria := map[string][]string{
		"object_type": {
			rbac.ResourceBucket.String(),
			rbac.ResourceFile.String(),
			rbac.ResourceFolder.String(),
			rbac.ResourceUser.String(),
		},
	}

	activities, err := s.ActivityLogger.Search(searchCriteria)
	if err != nil {
		return []map[string]interface{}{}
	}

	return activity.EnrichActivity(s.DB, activities)
}

func (s AdminService) GetBucketList(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
) []models.AdminBucketListItem {
	var buckets []models.Bucket
	s.DB.Find(&buckets)

	result := make([]models.AdminBucketListItem, 0, len(buckets))
	for _, bucket := range buckets {
		var creator models.User
		s.DB.Where("id = ?", bucket.CreatedBy).First(&creator)

		var memberCount int64
		s.DB.Model(&models.Membership{}).
			Where("bucket_id = ?", bucket.ID).
			Count(&memberCount)

		var fileCount int64
		s.DB.Model(&models.File{}).
			Where("bucket_id = ? AND status = ?", bucket.ID, models.FileStatusUploaded).
			Count(&fileCount)

		var size *int64
		s.DB.Model(&models.File{}).
			Where("bucket_id = ? AND status = ?", bucket.ID, models.FileStatusUploaded).
			Select("COALESCE(SUM(size), 0)").
			Scan(&size)

		item := models.AdminBucketListItem{
			ID:          bucket.ID,
			Name:        bucket.Name,
			CreatedAt:   bucket.CreatedAt,
			UpdatedAt:   bucket.UpdatedAt,
			Creator:     creator.ToActivity(),
			MemberCount: memberCount,
			FileCount:   fileCount,
		}

		if size != nil {
			item.Size = *size
		}

		result = append(result, item)
	}

	return result
}
