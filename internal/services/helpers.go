package services

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func loadUser(db *gorm.DB, userID uuid.UUID) (models.User, error) {
	var user models.User
	result := db.Where("id = ?", userID).First(&user)
	if result.RowsAffected == 0 {
		return models.User{}, apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
	}
	return user, nil
}
