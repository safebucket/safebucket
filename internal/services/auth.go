package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	"github.com/safebucket/safebucket/internal/rbac"
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
	r.With(m.Validate[models.AuthLoginBody]).Post("/login", s.loginHandler())
	r.With(m.Validate[models.AuthVerifyBody]).Post("/verify", handlers.CreateHandler(s.Verify))
	r.Post("/refresh", s.refreshHandler())
	r.Post("/logout", s.logoutHandler())

	r.Route("/mfa", func(r chi.Router) {
		r.With(m.Validate[models.MFALoginVerifyBody]).
			Post("/verify", s.mfaVerifyHandler())
	})
	r.Get("/me", handlers.GetOneHandler(s.Me))

	r.Mount("/reset-password", NewAuthPasswordResetService(s).Routes())

	r.Route("/providers", func(r chi.Router) {
		r.Get("/", handlers.GetListHandler(s.GetProviderList))
		r.Route("/{provider}", func(r chi.Router) {
			r.Get("/begin", handlers.OpenIDBeginHandler(s.OpenIDBegin))
			r.Get(
				"/callback",
				handlers.OpenIDCallbackHandler(s.AuthConfig.WebURL, s.AuthConfig.CookieSecureForce, s.OpenIDCallback),
			)
		})
	})
	return r
}

func (s AuthService) Login(
	isSecure bool,
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthLoginBody,
) (handlers.AuthFlowResult, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	var searchUser models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("email = ? AND provider_type = ? AND provider_key = ?",
			body.Email, models.LocalProviderType, string(models.LocalProviderType)).
		First(&searchUser)

	if result.RowsAffected != 1 {
		return handlers.AuthFlowResult{}, errors.New("invalid email / password combination")
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
	if err != nil || !match {
		return handlers.AuthFlowResult{}, errors.New("invalid email / password combination")
	}

	verifiedDevices := searchUser.GetVerifiedDevices()
	hasMFA := len(verifiedDevices) > 0

	if hasMFA || s.AuthConfig.MFARequired {
		restrictedToken, mfaErr := mfa.HandleMFARequired(logger, s.AuthConfig, &searchUser)
		if mfaErr != nil {
			return handlers.AuthFlowResult{}, mfaErr
		}
		return handlers.AuthFlowResult{
			Status:  http.StatusOK,
			Body:    models.AuthLoginResponse{MFARequired: true},
			Cookies: handlers.BuildMFACookie(isSecure, restrictedToken),
		}, nil
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &searchUser)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	if err = cache.CreateSession(s.Cache, searchUser.ID.String(), sid); err != nil {
		logger.Error("Failed to create session", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.ErrInternalServer
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  searchUser.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.UserLoggedIn,
			UserID:       searchUser.ID.String(),
			ObjectType:   rbac.ResourceUser.String(),
			ProviderType: string(models.LocalProviderType),
			ProviderName: s.Providers[string(models.LocalProviderType)].Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return handlers.AuthFlowResult{
		Status: http.StatusOK,
		Body:   models.AuthLoginResponse{},
		Cookies: handlers.BuildAuthCookies(
			isSecure,
			tokens.AccessToken,
			tokens.RefreshToken,
			string(models.LocalProviderType),
		),
	}, nil
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
	claims, err := h.ParseToken(s.AuthConfig.TokenSecret, body.AccessToken, true)
	if err != nil {
		return models.UserClaims{}, errors.New("invalid access token")
	}

	if claims.AudienceString() != configuration.AudienceAccessToken {
		return models.UserClaims{}, errors.New("invalid access token audience")
	}

	return claims, nil
}

func (s AuthService) Refresh(logger *zap.Logger, refreshTokenStr string) (string, error) {
	refreshToken, err := h.ParseRefreshToken(s.AuthConfig.TokenSecret, refreshTokenStr)
	if err != nil {
		return "", err
	}

	if refreshToken.SID == "" {
		return "", apierrors.NewAPIError(401, "SESSION_REVOKED")
	}

	maxAge := time.Duration(configuration.RefreshTokenExpiry) * time.Minute
	active, sessionErr := cache.IsSessionActive(s.Cache, refreshToken.UserID.String(), refreshToken.SID, maxAge)
	if sessionErr != nil {
		logger.Error("Session check failed during refresh", zap.Error(sessionErr))
		return "", apierrors.ErrInternalServer
	}
	if !active {
		return "", apierrors.NewAPIError(401, "SESSION_REVOKED")
	}

	var user models.User
	result := s.DB.Where("id = ?", refreshToken.UserID).First(&user)
	if result.RowsAffected == 0 {
		logger.Warn("User not found during token refresh",
			zap.String("user_id", refreshToken.UserID.String()))
		return "", apierrors.NewAPIError(401, "USER_NOT_FOUND")
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.TokenSecret,
		&user,
		refreshToken.Provider,
		refreshToken.SID,
	)
	if err != nil {
		return "", apierrors.ErrGenerateAccessTokenFailed
	}

	return accessToken, nil
}

func (s AuthService) loginHandler() http.HandlerFunc {
	return handlers.AuthFlowHandler(s.AuthConfig.CookieSecureForce, s.Login)
}

func (s AuthService) mfaVerifyHandler() http.HandlerFunc {
	return handlers.AuthFlowHandler(s.AuthConfig.CookieSecureForce, s.VerifyMFALogin)
}

func (s AuthService) refreshHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := m.GetLogger(r)

		var refreshTokenStr string
		if cookie, err := r.Cookie("safebucket_refresh_token"); err == nil {
			refreshTokenStr = cookie.Value
		} else {
			if tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				refreshTokenStr = tok
			}
		}

		if refreshTokenStr == "" {
			h.RespondWithError(w, http.StatusUnauthorized, []string{apierrors.CodeForbidden})
			return
		}

		newAccessToken, err := s.Refresh(logger, refreshTokenStr)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{err.Error()})
			} else {
				h.RespondWithError(w, http.StatusUnauthorized, []string{err.Error()})
			}
			return
		}

		handlers.SetAccessCookie(w, r, newAccessToken, s.AuthConfig.CookieSecureForce)
		h.RespondWithJSON(w, http.StatusOK, struct{}{})
	}
}

func (s AuthService) logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := m.GetLogger(r)
		claims, _ := h.GetUserClaims(r.Context())
		ids, ok := h.ParseUUIDs(w, r)
		if !ok {
			return
		}

		err := s.Logout(logger, claims, ids)
		handlers.ClearAuthCookies(w)
		if err != nil {
			var apiErr *apierrors.APIError
			if errors.As(err, &apiErr) {
				h.RespondWithError(w, apiErr.Code, []string{err.Error()})
			} else {
				h.RespondWithError(w, http.StatusInternalServerError, []string{err.Error()})
			}
			return
		}
		h.RespondWithJSON(w, http.StatusNoContent, nil)
	}
}

func (s AuthService) Me(
	_ *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
) (models.AuthMeResponse, error) {
	var count int64
	if err := s.DB.Model(&models.MFADevice{}).
		Where("user_id = ? AND is_verified = ?", claims.UserID, true).
		Count(&count).Error; err != nil {
		return models.AuthMeResponse{}, apierrors.ErrInternalServer
	}

	return models.AuthMeResponse{
		UserID:          claims.UserID,
		Email:           claims.Email,
		Role:            string(claims.Role),
		AuthProvider:    claims.Provider,
		MFA:             claims.MFA,
		MFADevicesCount: int(count),
	}, nil
}

func (s AuthService) VerifyMFALogin(
	isSecure bool,
	logger *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
	body models.MFALoginVerifyBody,
) (handlers.AuthFlowResult, error) {
	var user models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ? AND provider_type = ?", claims.UserID, models.LocalProviderType).
		First(&user)
	if result.RowsAffected == 0 {
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	verifiedDevices := user.GetVerifiedDevices()
	if len(verifiedDevices) == 0 {
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(400, "MFA_NOT_ENABLED")
	}

	attempts, err := cache.GetMFAAttempts(s.Cache, user.ID.String())
	if err != nil {
		logger.Error("Rate limit check failed - denying request", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(503, "SERVICE_UNAVAILABLE")
	}
	if attempts >= configuration.MFAMaxAttempts {
		logger.Warn("MFA rate limit exceeded",
			zap.String("user_id", user.ID.String()),
			zap.Int("attempts", attempts))
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(429, "MFA_RATE_LIMITED")
	}

	secret, deviceID, targetDevice, err := s.getMFASecretAndDevice(logger, &user, verifiedDevices, body.DeviceID)
	if err != nil {
		return handlers.AuthFlowResult{}, err
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
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
	}

	unused, err := cache.MarkTOTPCodeUsed(s.Cache, deviceID, body.Code)
	if err != nil {
		logger.Error("Failed to atomically check/mark TOTP code", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
	}

	if !unused {
		logger.Warn("TOTP code replay attempt detected",
			zap.String("device_id", deviceID))
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(401, "INVALID_MFA_CODE")
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
			s.AuthConfig.TokenSecret,
			&user,
			configuration.AudienceMFAReset,
			true,
			claims.ChallengeID,
		)
		if err != nil {
			return handlers.AuthFlowResult{}, apierrors.NewAPIError(500, "TOKEN_GENERATION_FAILED")
		}

		return handlers.AuthFlowResult{
			Status:  http.StatusOK,
			Body:    models.AuthLoginResponse{},
			Cookies: handlers.BuildMFACookie(isSecure, restrictedToken),
		}, nil
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &user)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	if sessionErr := cache.CreateSession(s.Cache, user.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return handlers.AuthFlowResult{}, apierrors.ErrInternalServer
	}

	return handlers.AuthFlowResult{
		Status: http.StatusOK,
		Body:   models.AuthLoginResponse{},
		Cookies: handlers.BuildAuthCookies(
			isSecure,
			tokens.AccessToken,
			tokens.RefreshToken,
			string(models.LocalProviderType),
		),
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

	sid := uuid.New().String()
	if sessionErr := cache.CreateSession(s.Cache, searchUser.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.TokenSecret,
		&searchUser,
		providerKey,
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.TokenSecret,
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
			ObjectType:   rbac.ResourceUser.String(),
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
