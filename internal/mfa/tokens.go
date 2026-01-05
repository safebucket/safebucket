package mfa

import (
	apierrors "api/internal/errors"
	h "api/internal/helpers"
	"api/internal/models"

	"go.uber.org/zap"
)

// HandleMFALogin generates MFA token and device list for MFA verification.
func HandleMFALogin(
	logger *zap.Logger,
	authConfig models.AuthConfig,
	user *models.User,
	verifiedDevices []models.MFADevice,
) (models.AuthLoginResponse, error) {
	mfaToken, err := h.NewMFAToken(authConfig.JWTSecret, user, authConfig.MFATokenExpiry)
	if err != nil {
		logger.Error("Failed to generate MFA token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResponse{
		MFARequired: true,
		MFAToken:    mfaToken,
		Devices:     verifiedDevices,
	}, nil
}

// GenerateTokensWithMFASetupRequired generates only MFA token for users who need to set up MFA.
// Access and refresh tokens are NOT issued until MFA setup is complete (via VerifyDevice).
// This prevents token leakage before MFA is configured.
func GenerateTokensWithMFASetupRequired(
	logger *zap.Logger,
	authConfig models.AuthConfig,
	user *models.User,
) (models.AuthLoginResponse, error) {
	mfaToken, err := h.NewMFAToken(authConfig.JWTSecret, user, authConfig.MFATokenExpiry)
	if err != nil {
		logger.Error("Failed to generate MFA token for setup required", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResponse{
		MFASetupRequired: true,
		MFAToken:         mfaToken,
	}, nil
}

// GenerateTokens generates access and refresh tokens for the user.
func GenerateTokens(
	authConfig models.AuthConfig,
	user *models.User,
) (models.AuthLoginResponse, error) {
	accessToken, err := h.NewAccessToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		authConfig.AccessTokenExpiry,
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(
		authConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		authConfig.RefreshTokenExpiry,
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateRefreshTokenFailed
	}

	return models.AuthLoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
