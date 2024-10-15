package sql

import (
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetById[T any](db *gorm.DB, id string) (T, error) {
	objId, err := uuid.Parse(id)
	if err != nil {
		return *new(T), errors.New("INVALID_ID")
	}

	var obj T
	result := db.Where("id = ?", objId).First(&obj)
	if result.RowsAffected == 0 {
		return *new(T), errors.New("NOT_FOUND")
	}

	return obj, nil
}

func Create[T any](db *gorm.DB, obj T) error {
	res := db.Create(&obj)

	if res.Error != nil {
		return errors.New("CREATE_FAILED")
	}

	return nil
}
