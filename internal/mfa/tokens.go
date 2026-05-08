package mfa

import (
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func HandleMFARequired(
	logger *zap.Logger,
	authConfig models.AuthConfig,
	user *models.User,
) (models.AuthLoginResult, error) {
	restrictedToken, err := h.NewRestrictedAccessToken(
		authConfig.JWTSecret,
		user,
		configuration.AudienceMFALogin,
		false,
		nil,
	)
	if err != nil {
		logger.Error("Failed to generate restricted access token", zap.Error(err))
		return models.AuthLoginResult{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResult{
		AuthLoginResponse: models.AuthLoginResponse{MFARequired: true},
		AccessToken:       restrictedToken,
	}, nil
}

func GenerateTokens(
	authConfig models.AuthConfig,
	user *models.User,
) (string, models.AuthLoginResult, error) {
	sid := uuid.New().String()

	accessToken, err := h.NewAccessToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		return "", models.AuthLoginResult{}, apierrors.ErrGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		return "", models.AuthLoginResult{}, apierrors.ErrGenerateRefreshTokenFailed
	}

	return sid, models.AuthLoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
