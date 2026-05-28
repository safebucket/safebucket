package services

import (
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func loadUser(tx *gorm.DB, userID uuid.UUID, user *models.User) error {
	result := tx.Where("id = ?", userID).Find(user)
	if result.RowsAffected == 0 {
		return apierrors.NewAPIError(404, apierrors.CodeUserNotFound)
	}
	return nil
}
