package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/mfa"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/alexedwards/argon2id"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
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
	r.Post("/logout", handlers.DeleteHandler(s.Logout))

	r.Route("/mfa", func(r chi.Router) {
		r.With(m.Validate[models.MFALoginVerifyBody]).
			Post("/verify", handlers.CreateHandler(s.VerifyMFALogin))
	})

	r.Mount("/reset-password", NewAuthPasswordResetService(s).Routes())

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

	// If user has MFA enabled OR MFA is required by admin, return restricted token
	if hasMFA || s.AuthConfig.MFARequired {
		return mfa.HandleMFARequired(logger, s.AuthConfig, &searchUser)
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &searchUser)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	if err = cache.CreateSession(s.Cache, searchUser.ID.String(), sid); err != nil {
		logger.Error("Failed to create session", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.ErrInternalServer
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  searchUser.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.UserLoggedIn,
			UserID:       searchUser.ID.String(),
			ObjectType:   "user",
			ProviderType: string(models.LocalProviderType),
			ProviderName: s.Providers[string(models.LocalProviderType)].Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return tokens, nil
}

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
	claims, err := h.ParseToken(s.AuthConfig.JWTSecret, body.AccessToken, true)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid access token")
	}

	if claims.AudienceString() != configuration.AudienceAccessToken {
		return models.UserClaims{}, errors.New("invalid access token audience")
	}

	return claims, nil
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

	if refreshToken.SID == "" {
		return models.AuthRefreshResponse{}, apierrors.NewAPIError(401, "SESSION_REVOKED")
	}

	maxAge := time.Duration(configuration.RefreshTokenExpiry) * time.Minute
	active, sessionErr := cache.IsSessionActive(s.Cache, refreshToken.UserID.String(), refreshToken.SID, maxAge)
	if sessionErr != nil {
		logger.Error("Session check failed during refresh", zap.Error(sessionErr))
		return models.AuthRefreshResponse{}, apierrors.ErrInternalServer
	}
	if !active {
		return models.AuthRefreshResponse{}, apierrors.NewAPIError(401, "SESSION_REVOKED")
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
		refreshToken.SID,
	)
	if err != nil {
		return models.AuthRefreshResponse{}, apierrors.ErrGenerateAccessTokenFailed
	}

	return models.AuthRefreshResponse{AccessToken: accessToken}, nil
}

func (s AuthService) VerifyMFALogin(
	logger *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
	body models.MFALoginVerifyBody,
) (models.AuthLoginResponse, error) {
	var user models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ? AND provider_type = ?", claims.UserID, models.LocalProviderType).
		First(&user)
	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	verifiedDevices := user.GetVerifiedDevices()
	if len(verifiedDevices) == 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "MFA_NOT_ENABLED")
	}

	attempts, err := cache.GetMFAAttempts(s.Cache, user.ID.String())
	if err != nil {
		logger.Error("Rate limit check failed - denying request", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(503, "SERVICE_UNAVAILABLE")
	}
	if attempts >= configuration.MFAMaxAttempts {
		logger.Warn("MFA rate limit exceeded",
			zap.String("user_id", user.ID.String()),
			zap.Int("attempts", attempts))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(429, "MFA_RATE_LIMITED")
	}

	secret, deviceID, targetDevice, err := s.getMFASecretAndDevice(logger, &user, verifiedDevices, body.DeviceID)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	if targetDevice != nil {
		s.DB.Model(targetDevice).Update("last_used_at", time.Now())
	}

	if !h.ValidateTOTPCode(secret, body.Code) {
		if incErr := cache.IncrementMFAAttempts(s.Cache, user.ID.String()); incErr != nil {
			logger.Error("Failed to increment MFA attempts", zap.Error(incErr))
		}
		logger.Warn("MFA verification failed",
			zap.String("user_id", user.ID.String()),
			zap.String("device_id", deviceID),
			zap.String("email", user.Email))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
	}

	unused, err := cache.MarkTOTPCodeUsed(s.Cache, deviceID, body.Code)
	if err != nil {
		logger.Error("Failed to atomically check/mark TOTP code", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
	}

	if !unused {
		logger.Warn("TOTP code replay attempt detected",
			zap.String("device_id", deviceID))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
	}

	if resetErr := cache.ResetMFAAttempts(s.Cache, user.ID.String()); resetErr != nil {
		logger.Warn("Failed to reset MFA attempts", zap.Error(resetErr))
	}

	logger.Info("MFA login verification successful",
		zap.String("user_id", user.ID.String()),
		zap.String("device_id", deviceID),
		zap.String("email", user.Email))

	if claims.AudienceString() == configuration.AudienceMFAReset {
		var restrictedToken string
		restrictedToken, err = h.NewRestrictedAccessToken(
			s.AuthConfig.JWTSecret,
			&user,
			configuration.AudienceMFAReset,
			true,
			claims.ChallengeID,
		)
		if err != nil {
			return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "TOKEN_GENERATION_FAILED")
		}

		return models.AuthLoginResponse{
			AccessToken: restrictedToken,
			MFARequired: false,
		}, nil
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &user)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	if sessionErr := cache.CreateSession(s.Cache, user.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return models.AuthLoginResponse{}, apierrors.ErrInternalServer
	}

	return tokens, nil
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

	sid := uuid.New().String()
	if sessionErr := cache.CreateSession(s.Cache, searchUser.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		&searchUser,
		providerKey,
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		&searchUser,
		providerKey,
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  searchUser.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.UserLoggedIn,
			UserID:       searchUser.ID.String(),
			ObjectType:   "user",
			ProviderType: string(models.OIDCProviderType),
			ProviderName: provider.Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return accessToken, refreshToken, nil
}

func (s AuthService) Logout(
	logger *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
) error {
	if claims.SID == "" {
		return apierrors.NewAPIError(401, "SESSION_REVOKED")
	}
	if err := cache.RevokeSession(s.Cache, claims.UserID.String(), claims.SID); err != nil {
		logger.Error("Failed to revoke session", zap.Error(err))
		return apierrors.ErrInternalServer
	}

	return nil
}
