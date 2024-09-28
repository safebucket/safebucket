package services

import (
	c "api/internal/common"
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
	r.Get("/", c.GetListHandler(s.GetBucketList))
	r.With(c.Validate[models.Bucket]).Post("/", c.CreateHandler(s.CreateBucket))

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", c.GetOneHandler(s.GetBucket))
		r.With(c.Validate[models.Bucket]).Patch("/", c.UpdateHandler(s.UpdateBucket))
		r.Delete("/", c.DeleteHandler(s.DeleteBucket))
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

func (s BucketService) GetBucket(id string) (models.Bucket, error) {
	var bucket models.Bucket
	_, err := uuid.Parse(id)
	if err != nil {
		return models.Bucket{}, errors.New("invalid ID")
	}
	result := s.DB.Where("id = ?", id).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (s BucketService) UpdateBucket(id string, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: id}
	_, err := uuid.Parse(id)
	if err != nil {
		return models.Bucket{}, errors.New("invalid ID")
	}
	result := s.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (s BucketService) DeleteBucket(id string) error {
	result := s.DB.Where("id = ?", id).Delete(&models.Bucket{})
	_, err := uuid.Parse(id)
	if err != nil {
		return errors.New("invalid ID")
	}
	if result.RowsAffected == 0 {
		return errors.New("bucket not found")
	} else {
		return nil
	}
}
