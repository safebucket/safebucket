package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api/internal/activity"
	"api/internal/cache"
	"api/internal/configuration"
	apierrors "api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
	"api/internal/mfa"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/sql"

	"github.com/alexedwards/argon2id"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthService struct {
	DB             *gorm.DB
	Cache          cache.ICache
	AuthConfig     models.AuthConfig
	Providers      configuration.Providers
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
}

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(m.Validate[models.AuthLoginBody]).Post("/login", handlers.CreateHandler(s.Login))
	r.With(m.Validate[models.AuthVerifyBody]).Post("/verify", handlers.CreateHandler(s.Verify))
	r.With(m.Validate[models.AuthRefreshBody]).Post("/refresh", handlers.CreateHandler(s.Refresh))

	r.Route("/mfa", func(r chi.Router) {
		r.With(m.Validate[models.MFALoginVerifyBody]).
			Post("/verify", handlers.CreateHandler(s.VerifyMFALogin))
	})

	r.Route("/reset-password", func(r chi.Router) {
		r.With(m.Validate[models.PasswordResetRequestBody]).
			Post("/", handlers.CreateHandler(s.RequestPasswordReset))
		r.Route("/{id0}", func(r chi.Router) {
			r.With(m.Validate[models.PasswordResetValidateBody]).
				Post("/validate", handlers.CreateHandler(s.ValidatePasswordReset))
		})
	})

	r.Route("/providers", func(r chi.Router) {
		r.Get("/", handlers.GetListHandler(s.GetProviderList))
		r.Route("/{provider}", func(r chi.Router) {
			r.Get("/begin", handlers.OpenIDBeginHandler(s.OpenIDBegin))
			r.Get("/callback", handlers.OpenIDCallbackHandler(s.AuthConfig.WebURL, s.OpenIDCallback))
		})
	})
	return r
}

func (s AuthService) Login(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthLoginBody,
) (models.AuthLoginResponse, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return models.AuthLoginResponse{}, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	// Load user with verified MFA devices
	var searchUser models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("email = ? AND provider_type = ? AND provider_key = ?",
			body.Email, models.LocalProviderType, string(models.LocalProviderType)).
		First(&searchUser)

	if result.RowsAffected != 1 {
		return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
	if err != nil || !match {
		return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
	}

	verifiedDevices := searchUser.GetVerifiedDevices()
	hasMFA := len(verifiedDevices) > 0

	if hasMFA {
		return mfa.HandleMFALogin(logger, s.AuthConfig, &searchUser, verifiedDevices)
	}

	// Check if MFA is required but not set up for this user
	if s.AuthConfig.MFARequired {
		return mfa.GenerateTokensWithMFASetupRequired(logger, s.AuthConfig, &searchUser)
	}

	tokens, err := mfa.GenerateTokens(s.AuthConfig, &searchUser)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  searchUser.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":        activity.UserLoggedIn,
			"user_id":       searchUser.ID.String(),
			"object_type":   "user",
			"provider_type": string(models.LocalProviderType),
			"provider_name": s.Providers[string(models.LocalProviderType)].Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return tokens, nil
}

// getMFASecretAndDevice retrieves the MFA secret and device ID for verification.
// Returns (secret, deviceID, targetDevice, error).
func (s AuthService) getMFASecretAndDevice(
	logger *zap.Logger,
	user *models.User,
	verifiedDevices []models.MFADevice,
	requestedDeviceID *uuid.UUID,
) (string, string, *models.MFADevice, error) {
	if len(verifiedDevices) == 0 {
		return "", "", nil, apierrors.NewAPIError(400, "MFA_NOT_ENABLED")
	}

	targetDevice, err := s.selectMFADevice(user, verifiedDevices, requestedDeviceID)
	if err != nil {
		return "", "", nil, err
	}

	secret, err := h.DecryptSecret(targetDevice.EncryptedSecret, []byte(s.AuthConfig.MFAEncryptionKey))
	if err != nil {
		logger.Error("Failed to decrypt TOTP secret", zap.Error(err))
		return "", "", nil, apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
	}

	return secret, targetDevice.ID.String(), targetDevice, nil
}

// selectMFADevice selects the MFA device for verification.
func (s AuthService) selectMFADevice(
	user *models.User,
	verifiedDevices []models.MFADevice,
	requestedDeviceID *uuid.UUID,
) (*models.MFADevice, error) {
	if requestedDeviceID != nil {
		for i := range verifiedDevices {
			if verifiedDevices[i].ID == *requestedDeviceID {
				return &verifiedDevices[i], nil
			}
		}
		return nil, apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
	}

	// Use default device or first verified device
	if device := user.GetDefaultDevice(); device != nil {
		return device, nil
	}

	if len(verifiedDevices) > 0 {
		return &verifiedDevices[0], nil
	}

	return nil, apierrors.NewAPIError(400, "MFA_NOT_ENABLED")
}

func (s AuthService) Verify(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthVerifyBody,
) (any, error) {
	data, err := h.ParseAccessToken(s.AuthConfig.JWTSecret, body.AccessToken)
	return data, err
}

func (s AuthService) Refresh(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthRefreshBody,
) (models.AuthRefreshResponse, error) {
	refreshToken, err := h.ParseRefreshToken(s.AuthConfig.JWTSecret, body.RefreshToken)
	if err != nil {
		return models.AuthRefreshResponse{}, err
	}

	var user models.User
	result := s.DB.Where("id = ?", refreshToken.UserID).First(&user)
	if result.RowsAffected == 0 {
		logger.Warn("User not found during token refresh",
			zap.String("user_id", refreshToken.UserID.String()))
		return models.AuthRefreshResponse{}, apierrors.NewAPIError(401, "USER_NOT_FOUND")
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		&user,
		refreshToken.Provider,
		s.AuthConfig.AccessTokenExpiry,
	)
	if err != nil {
		return models.AuthRefreshResponse{}, apierrors.ErrGenerateAccessTokenFailed
	}

	return models.AuthRefreshResponse{AccessToken: accessToken}, nil
}

// VerifyMFALogin verifies TOTP code during login and issues access/refresh tokens.
func (s AuthService) VerifyMFALogin(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.MFALoginVerifyBody,
) (models.AuthLoginResponse, error) {
	mfaClaims, err := h.ParseMFAToken(s.AuthConfig.JWTSecret, body.MFAToken)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "INVALID_MFA_TOKEN")
	}

	// Load user with verified MFA devices
	var user models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ? AND provider_type = ?", mfaClaims.UserID, models.LocalProviderType).
		First(&user)
	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	verifiedDevices := user.GetVerifiedDevices()
	if len(verifiedDevices) == 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "MFA_NOT_ENABLED")
	}

	// Check MFA rate limiting
	attempts, err := s.Cache.GetMFAAttempts(user.ID.String())
	if err != nil {
		logger.Error("Failed to get MFA attempts", zap.Error(err))
	}
	if attempts >= configuration.MFAMaxAttempts {
		logger.Warn("MFA rate limit exceeded",
			zap.String("user_id", user.ID.String()),
			zap.Int("attempts", attempts))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(429, "MFA_RATE_LIMITED")
	}

	// Get secret and device ID for verification
	secret, deviceID, targetDevice, err := s.getMFASecretAndDevice(logger, &user, verifiedDevices, body.DeviceID)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	// Update last_used_at for device-based verification
	if targetDevice != nil {
		s.DB.Model(targetDevice).Update("last_used_at", time.Now())
	}

	// Validate TOTP code
	if !h.ValidateTOTPCode(secret, body.Code) {
		// Increment failed attempts
		if incErr := s.Cache.IncrementMFAAttempts(user.ID.String()); incErr != nil {
			logger.Error("Failed to increment MFA attempts", zap.Error(incErr))
		}
		logger.Warn("MFA login verification failed",
			zap.String("user_id", user.ID.String()),
			zap.String("device_id", deviceID),
			zap.String("email", user.Email))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
	}

	// Check replay protection (per device or user for legacy)
	used, err := s.Cache.IsTOTPCodeUsed(deviceID, body.Code)
	if err != nil {
		logger.Error("Failed to check TOTP code usage", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
	}
	if used {
		logger.Warn("TOTP code replay attempt detected",
			zap.String("device_id", deviceID))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
	}

	if err = s.Cache.MarkTOTPCodeUsed(deviceID, body.Code); err != nil {
		logger.Error("Failed to mark TOTP code as used", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
	}

	// Reset MFA attempts on successful verification
	if resetErr := s.Cache.ResetMFAAttempts(user.ID.String()); resetErr != nil {
		logger.Warn("Failed to reset MFA attempts", zap.Error(resetErr))
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		&user,
		string(models.LocalProviderType),
		s.AuthConfig.AccessTokenExpiry,
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		&user,
		string(models.LocalProviderType),
		s.AuthConfig.RefreshTokenExpiry,
	)
	if err != nil {
		return models.AuthLoginResponse{}, apierrors.ErrGenerateRefreshTokenFailed
	}

	logger.Info("MFA login verification successful",
		zap.String("user_id", user.ID.String()),
		zap.String("device_id", deviceID),
		zap.String("email", user.Email))

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s AuthService) GetProviderList(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
) []models.ProviderResponse {
	providers := make([]models.ProviderResponse, len(s.Providers))
	for id, provider := range s.Providers {
		if len(provider.Domains) == 0 {
			provider.Domains = []string{}
		}

		providers[provider.Order] = models.ProviderResponse{
			ID:      id,
			Name:    provider.Name,
			Type:    provider.Type,
			Domains: provider.Domains,
		}
	}
	return providers
}

func (s AuthService) OpenIDBegin(providerName string, state string, nonce string) (string, error) {
	provider, ok := s.Providers[providerName]
	if !ok {
		return "", errors.New("provider not found")
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, logger *zap.Logger, providerKey string, code string, nonce string,
) (string, string, error) {
	provider, ok := s.Providers[providerKey]
	if !ok {
		return "", "", errors.New("provider not found")
	}

	oauth2Token, err := provider.OauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange token %s", err.Error())
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", "", errors.New("no id_token field in oauth2 token")
	}

	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify ID token %s", err.Error())
	}

	if idToken.Nonce != nonce {
		return "", "", errors.New("nonce does not match")
	}

	userInfo, err := provider.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info %s", err.Error())
	}

	if !h.IsDomainAllowed(userInfo.Email, s.Providers[providerKey].Domains) {
		logger.Debug("Domain not allowed")
		return "", "", apierrors.NewAPIError(403, "FORBIDDEN")
	}

	searchUser := models.User{
		Email:        userInfo.Email,
		ProviderType: models.OIDCProviderType,
		ProviderKey:  providerKey,
	}
	result := s.DB.Where(searchUser, "email", "provider_type", "provider_key").Find(&searchUser)
	if result.RowsAffected == 0 {
		searchUser.Role = models.RoleUser

		err = sql.CreateUserWithInvites(logger, s.DB, &searchUser)
		if err != nil {
			logger.Error("Failed to create user with invites", zap.Error(err))
			return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		&searchUser,
		providerKey,
		s.AuthConfig.AccessTokenExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		&searchUser,
		providerKey,
		s.AuthConfig.RefreshTokenExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  searchUser.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":        activity.UserLoggedIn,
			"user_id":       searchUser.ID.String(),
			"object_type":   "user",
			"provider_type": string(models.OIDCProviderType),
			"provider_name": provider.Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return accessToken, refreshToken, nil
}

func (s AuthService) ValidatePasswordReset(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.PasswordResetValidateBody,
) (models.AuthLoginResponse, error) {
	challengeID := ids[0]

	var challenge models.Challenge
	var user *models.User

	// Use transaction with row-level locking to prevent race conditions
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("User").
			Where("id = ? AND type = ?", challengeID, models.ChallengeTypePasswordReset).
			First(&challenge)

		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
		}

		if challenge.User == nil {
			logger.Error("Challenge has no associated user")
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		user = challenge.User

		if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
			tx.Delete(&challenge)
			return apierrors.NewAPIError(410, "CHALLENGE_EXPIRED")
		}

		match, err := argon2id.ComparePasswordAndHash(
			strings.ToUpper(body.Code),
			challenge.HashedSecret,
		)
		if err != nil || !match {
			challenge.AttemptsLeft--

			if challenge.AttemptsLeft <= 0 {
				logger.Warn("Password reset challenge soft deleted due to too many failed attempts",
					zap.String("challenge_id", challenge.ID.String()),
					zap.String("user_id", challenge.UserID.String()),
					zap.Int("attempts_left", challenge.AttemptsLeft))
				tx.Delete(&challenge)

				return apierrors.NewAPIError(403, "CHALLENGE_LOCKED")
			}

			if updateErr := tx.Save(&challenge).Error; updateErr != nil {
				logger.Error("Failed to update attempts counter", zap.Error(updateErr))
				return updateErr
			}
			return apierrors.NewAPIError(401, "WRONG_CODE")
		}

		return nil
	})
	if err != nil {
		return models.AuthLoginResponse{}, err
	}
	hashedPassword, err := h.CreateHash(body.NewPassword)
	if err != nil {
		logger.Error("Failed to hash new password", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "PASSWORD_UPDATE_FAILED")
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(user).Update("hashed_password", hashedPassword)
		if updateResult.Error != nil {
			logger.Error("Failed to update password", zap.Error(updateResult.Error))
			return apierrors.NewAPIError(500, "PASSWORD_UPDATE_FAILED")
		}

		deleteResult := tx.Delete(&challenge)
		if deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			return apierrors.NewAPIError(500, "CHALLENGE_CLEANUP_FAILED")
		}

		return nil
	})
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	resetDate := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	successEvent := events.NewPasswordResetSuccess(
		s.Publisher,
		user.Email,
		s.AuthConfig.WebURL,
		resetDate,
	)
	successEvent.Trigger()

	action := models.Activity{
		Message: activity.PasswordResetCompleted,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":       activity.PasswordResetCompleted,
			"user_id":      user.ID.String(),
			"challenge_id": challengeID.String(),
			"object_type":  "user",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log password reset completion", zap.Error(logErr))
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		s.AuthConfig.AccessTokenExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(
			500,
			"GENERATE_ACCESS_TOKEN_FAILED",
		)
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
		s.AuthConfig.RefreshTokenExpiry,
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(
			500,
			"GENERATE_REFRESH_TOKEN_FAILED",
		)
	}

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s AuthService) RequestPasswordReset(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.PasswordResetRequestBody,
) (any, error) {
	var user models.User
	result := s.DB.Where("email = ? AND provider_type = ?", body.Email, models.LocalProviderType).
		First(&user)

	if result.RowsAffected == 0 {
		return nil, nil
	}

	secret, err := h.GenerateSecret()
	if err != nil {
		logger.Error("Failed to generate secret", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		logger.Error("Failed to hash secret", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	// Delete any existing password reset challenges for this user
	s.DB.Where("user_id = ? AND type = ?", user.ID, models.ChallengeTypePasswordReset).
		Delete(&models.Challenge{})

	// Create a new password reset challenge with configurable expiration
	expiresAt := time.Now().Add(configuration.SecurityChallengeExpirationMinutes * time.Minute)
	challenge := models.Challenge{
		Type:         models.ChallengeTypePasswordReset,
		UserID:       &user.ID,
		HashedSecret: hashedSecret,
		ExpiresAt:    &expiresAt,
		AttemptsLeft: configuration.SecurityChallengeMaxFailedAttempts,
	}

	result = s.DB.Create(&challenge)
	if result.Error != nil {
		logger.Error("Failed to create challenge", zap.Error(result.Error))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	// Send password reset email
	event := events.NewPasswordResetChallenge(
		s.Publisher,
		secret,
		user.Email,
		challenge.ID.String(),
		s.AuthConfig.WebURL,
	)
	event.Trigger()

	return nil, nil
}
