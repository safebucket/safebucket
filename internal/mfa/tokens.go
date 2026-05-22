package mfa

import (
	"net/http"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

func HandleMFARequired(
	logger *zap.Logger,
	authConfig models.AuthConfig,
	user *models.User,
) (string, error) {
	restrictedToken, err := h.NewRestrictedAccessToken(
		authConfig.TokenSecret,
		user,
		configuration.AudienceMFALogin,
		false,
		nil,
	)
	if err != nil {
		logger.Error("Failed to generate restricted access token", zap.Error(err))
		return "", apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	return restrictedToken, nil
}

func GenerateTokens(
	authConfig models.AuthConfig,
	user *models.User,
) (string, TokenPair, error) {
	sid := uuid.New().String()

	accessToken, err := h.NewAccessToken(
		authConfig.TokenSecret,
		user,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		return "", TokenPair{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeGenerateAccessTokenFailed)
	}

	refreshToken, err := h.NewRefreshToken(
		authConfig.TokenSecret,
		user,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		return "", TokenPair{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeGenerateRefreshTokenFailed)
	}

	return sid, TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
