package services

import (
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BucketService struct {
	DB *gorm.DB
}

func (s BucketService) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", handlers.GetListHandler(s.GetBucketList))
	r.With(h.Validate[models.Bucket]).Post("/", handlers.CreateHandler(s.CreateBucket))

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", handlers.GetOneHandler(s.GetBucket))
		r.With(h.Validate[models.Bucket]).Patch("/", handlers.UpdateHandler(s.UpdateBucket))
		r.Delete("/", handlers.DeleteHandler(s.DeleteBucket))
	})
	return r
}

func (s BucketService) CreateBucket(body models.Bucket) (models.Bucket, error) {
	s.DB.Create(&body)
	return body, nil
}

func (s BucketService) GetBucketList() []models.Bucket {
	var buckets []models.Bucket
	s.DB.Find(&buckets)
	return buckets
}

func (s BucketService) GetBucket(id uuid.UUID) (models.Bucket, error) {
	var bucket models.Bucket
	result := s.DB.Where("id = ?", id).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (s BucketService) UpdateBucket(id uuid.UUID, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: id}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (s BucketService) DeleteBucket(id uuid.UUID) error {
	result := s.DB.Where("id = ?", id).Delete(&models.Bucket{})
	if result.RowsAffected == 0 {
		return errors.New("bucket not found")
	} else {
		return nil
	}
}
