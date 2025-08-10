package sql

import (
	"api/internal/errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetById[T any](db *gorm.DB, id uuid.UUID) (T, error) {
	var obj T
	result := db.Where("id = ?", id).First(&obj)
	if result.RowsAffected == 0 {
		return *new(T), errors.NewAPIError(404, "NOT_FOUND")
	}

	return obj, nil
}

func Create[T any](db *gorm.DB, obj T) error {
	res := db.Create(&obj)

	if res.Error != nil {
		return errors.ErrorCreateFailed
	}

	return nil
}
