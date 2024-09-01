package repositories

import (
	c "api/internal/common"
	"api/internal/models"
	"errors"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type BucketRepo struct {
	DB *gorm.DB
}

func (br BucketRepo) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", c.GetListHandler[models.Bucket](br))
	r.With(c.Validate[models.Bucket]).Post("/", c.CreateHandler[models.Bucket](br))

	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", c.GetOneHandler[models.Bucket](br))
		r.With(c.Validate[models.Bucket]).Patch("/", c.UpdateHandler[models.Bucket](br))
		r.Delete("/", c.DeleteHandler[models.Bucket](br))
	})
	return r
}

func (br BucketRepo) Create(body models.Bucket) (models.Bucket, error) {
	br.DB.Create(&body)
	return body, nil
}

func (br BucketRepo) GetList() []models.Bucket {
	var buckets []models.Bucket
	br.DB.Find(&buckets)
	return buckets
}

func (br BucketRepo) GetOne(id uint) (models.Bucket, error) {
	var bucket models.Bucket
	result := br.DB.Where("id = ?", id).First(&bucket)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (br BucketRepo) Update(id uint, body models.Bucket) (models.Bucket, error) {
	bucket := models.Bucket{ID: id}
	result := br.DB.Model(&bucket).Updates(body)
	if result.RowsAffected == 0 {
		return bucket, errors.New("bucket not found")
	} else {
		return bucket, nil
	}
}

func (br BucketRepo) Delete(id uint) error {
	result := br.DB.Where("id = ?", id).Delete(&models.Bucket{})
	if result.RowsAffected == 0 {
		return errors.New("bucket not found")
	} else {
		return nil
	}
}
