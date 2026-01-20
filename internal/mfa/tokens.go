package mfa

import (
	"api/internal/configuration"
	apierrors "api/internal/errors"
	h "api/internal/helpers"
	"api/internal/models"

	"go.uber.org/zap"
)

// HandleMFARequired generates a restricted access token for MFA flows.
// Returns a token with limited access that can only be used for:
// - Listing MFA devices
// - Adding/verifying MFA devices (during setup)
// - Completing MFA verification
// Frontend determines setup vs verify state by checking if devices list is empty.
func HandleMFARequired(
	logger *zap.Logger,
	authConfig models.AuthConfig,
	user *models.User,
) (models.AuthLoginResponse, error) {
	restrictedToken, err := h.NewRestrictedAccessToken(
		authConfig.JWTSecret,
		user,
		configuration.AudienceMFALogin,
		false,
		nil,
	)
	if err != nil {
		logger.Error("Failed to generate restricted access token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResponse{
		AccessToken: restrictedToken,
		MFARequired: true,
	}, nil
}

// GenerateTokens generates full access and refresh tokens for the user.
// Used after successful MFA verification or for users without MFA.
func GenerateTokens(
	authConfig models.AuthConfig,
	user *models.User,
) (models.AuthLoginResponse, error) {
	accessToken, err := h.NewAccessToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateRefreshTokenFailed
	}

	return models.AuthLoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
