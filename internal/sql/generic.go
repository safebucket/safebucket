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
