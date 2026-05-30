package services

import (
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
)

func (s AuthService) authenticateLocal(body models.AuthLoginBody) (models.User, error) {
	var searchUser models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("email = ? AND provider_type = ? AND provider_key = ?",
			body.Email, models.LocalProviderType, string(models.LocalProviderType)).
		First(&searchUser)

	if result.RowsAffected != 1 {
		return models.User{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidCredentials)
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
	if err != nil || !match {
		return models.User{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidCredentials)
	}
	return searchUser, nil
}
