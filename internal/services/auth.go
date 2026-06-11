package services

import (
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
	"github.com/safebucket/safebucket/internal/tracing"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
			r.With(m.Validate[models.AuthLoginBody]).Post(
				"/login",
				handlers.AuthFlowProviderHandler(s.AuthConfig.CookieSecureForce, s.LDAPLogin),
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
	provider, ok := s.resolveAuthProvider(body.Email)
	if !ok {
		logger.Debug("No credential provider matches this email")
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	user, found, err := sql.FindUserByIdentityProvider(
		s.DB, body.Email, models.LocalProviderType, string(models.LocalProviderType), true,
	)
	if err != nil {
		logger.Error("Failed to look up user", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}
	if !found {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidCredentials)
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.HashedPassword)
	if err != nil || !match {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidCredentials)
	}

	return s.finalizeLogin(isSecure, logger, &user, provider.Type, provider.Name, provider.MFARequired)
}

func (s AuthService) finalizeLogin(
	isSecure bool,
	logger *zap.Logger,
	user *models.User,
	providerType models.ProviderType,
	providerName string,
	providerMFARequired bool,
) (handlers.AuthFlowResult, error) {
	verifiedDevices := user.GetVerifiedDevices()
	hasMFA := len(verifiedDevices) > 0

	if hasMFA || providerMFARequired {
		restrictedToken, mfaErr := mfa.HandleMFARequired(logger, s.AuthConfig, user)
		if mfaErr != nil {
			return handlers.AuthFlowResult{}, mfaErr
		}
		return handlers.AuthFlowResult{
			Status:  http.StatusOK,
			Body:    models.AuthLoginResponse{MFARequired: true},
			Cookies: handlers.BuildMFACookie(isSecure, restrictedToken),
		}, nil
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, user)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	if err = cache.CreateSession(s.Cache, user.ID.String(), sid); err != nil {
		logger.Error("Failed to create session", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.UserLoggedIn,
			UserID:       user.ID.String(),
			ObjectType:   rbac.ResourceUser.String(),
			ProviderType: string(providerType),
			ProviderName: providerName,
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
			user.ProviderKey,
		),
	}, nil
}

func (s AuthService) resolveAuthProvider(
	email string,
) (configuration.Provider, bool) {
	for _, provider := range s.Providers {
		if provider.Type != models.LocalProviderType {
			continue
		}
		if h.IsDomainAllowed(email, provider.Domains) {
			return provider, true
		}
	}
	return configuration.Provider{}, false
}

func (s AuthService) getMFASecretAndDevice(
	logger *zap.Logger,
	user *models.User,
	verifiedDevices []models.MFADevice,
	requestedDeviceID *uuid.UUID,
) (string, string, *models.MFADevice, error) {
	if len(verifiedDevices) == 0 {
		return "", "", nil, apierrors.New(http.StatusBadRequest, apierrors.CodeMFANotEnabled)
	}

	targetDevice, err := s.selectMFADevice(user, verifiedDevices, requestedDeviceID)
	if err != nil {
		return "", "", nil, err
	}

	secret, err := h.DecryptSecret(targetDevice.EncryptedSecret, []byte(s.AuthConfig.MFAEncryptionKey))
	if err != nil {
		logger.Error("Failed to decrypt TOTP secret", zap.Error(err))
		return "", "", nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeMFAVerificationFailed)
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
		return nil, apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
	}

	if device := user.GetDefaultDevice(); device != nil {
		return device, nil
	}

	if len(verifiedDevices) > 0 {
		return &verifiedDevices[0], nil
	}

	return nil, apierrors.New(http.StatusBadRequest, apierrors.CodeMFANotEnabled)
}

func (s AuthService) Verify(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthVerifyBody,
) (any, error) {
	claims, err := h.ParseToken(s.AuthConfig.TokenSecret, body.AccessToken, true)
	if err != nil {
		return models.UserClaims{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidAccessToken)
	}

	if claims.AudienceString() != configuration.AudienceAccessToken {
		return models.UserClaims{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidAccessTokenAudience)
	}

	return claims, nil
}

func (s AuthService) Refresh(logger *zap.Logger, refreshTokenStr string) (string, error) {
	refreshToken, err := h.ParseRefreshToken(s.AuthConfig.TokenSecret, refreshTokenStr)
	if err != nil {
		return "", apierrors.New(http.StatusUnauthorized, apierrors.CodeUnauthorized)
	}

	if refreshToken.SID == "" {
		return "", apierrors.New(http.StatusUnauthorized, apierrors.CodeSessionRevoked)
	}

	maxAge := time.Duration(configuration.RefreshTokenExpiry) * time.Minute
	active, sessionErr := cache.IsSessionActive(s.Cache, refreshToken.UserID.String(), refreshToken.SID, maxAge)
	if sessionErr != nil {
		logger.Error("Session check failed during refresh", zap.Error(sessionErr))
		return "", apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	if !active {
		return "", apierrors.New(http.StatusUnauthorized, apierrors.CodeSessionRevoked)
	}

	var user models.User
	result := s.DB.Where("id = ?", refreshToken.UserID).First(&user)
	if result.RowsAffected == 0 {
		logger.Warn("User not found during token refresh",
			zap.String("user_id", refreshToken.UserID.String()))
		return "", apierrors.New(http.StatusUnauthorized, apierrors.CodeUserNotFound)
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.TokenSecret,
		&user,
		refreshToken.Provider,
		refreshToken.SID,
	)
	if err != nil {
		return "", apierrors.New(http.StatusInternalServerError, apierrors.CodeGenerateAccessTokenFailed)
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
		ctx, span := tracing.StartSpan(r.Context(), "handlers.Refresh")
		defer span.End()
		r = r.WithContext(ctx)

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
			handlers.WriteError(span, w, err)
			return
		}

		handlers.SetAccessCookie(w, r, newAccessToken, s.AuthConfig.CookieSecureForce)
		h.RespondWithJSON(w, http.StatusOK, struct{}{})
	}
}

func (s AuthService) logoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), "handlers.Logout")
		defer span.End()
		r = r.WithContext(ctx)

		logger := m.GetLogger(r)
		handlers.ClearAuthCookies(w)

		refreshCookie, cookieErr := r.Cookie(configuration.CookieRefreshToken)
		if cookieErr != nil {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
			return
		}

		claims, err := h.ParseRefreshToken(s.AuthConfig.TokenSecret, refreshCookie.Value)
		if err != nil {
			h.RespondWithJSON(w, http.StatusNoContent, nil)
			return
		}

		if logoutErr := s.Logout(logger, claims); logoutErr != nil {
			handlers.WriteError(span, w, logoutErr)
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
		return models.AuthMeResponse{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
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
		Where("id = ?", claims.UserID).
		First(&user)
	if result.RowsAffected == 0 {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
	}

	verifiedDevices := user.GetVerifiedDevices()
	if len(verifiedDevices) == 0 {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusBadRequest, apierrors.CodeMFANotEnabled)
	}

	attempts, err := cache.GetMFAAttempts(s.Cache, user.ID.String())
	if err != nil {
		logger.Error("Rate limit check failed - denying request", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusServiceUnavailable, apierrors.CodeServiceUnavailable)
	}
	if attempts >= configuration.MFAMaxAttempts {
		logger.Warn("MFA rate limit exceeded",
			zap.String("user_id", user.ID.String()),
			zap.Int("attempts", attempts))
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusTooManyRequests, apierrors.CodeMFARateLimited)
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
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
	}

	unused, err := cache.MarkTOTPCodeUsed(s.Cache, deviceID, body.Code)
	if err != nil {
		logger.Error("Failed to atomically check/mark TOTP code", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeMFAVerificationFailed,
		)
	}

	if !unused {
		logger.Warn("TOTP code replay attempt detected",
			zap.String("device_id", deviceID))
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
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
			return handlers.AuthFlowResult{}, apierrors.New(
				http.StatusInternalServerError,
				apierrors.CodeTokenGenerationFailed,
			)
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
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	action := models.Activity{
		Message: activity.UserLoggedIn,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.UserLoggedIn,
			UserID:       user.ID.String(),
			ObjectType:   rbac.ResourceUser.String(),
			ProviderType: string(user.ProviderType),
			ProviderName: s.Providers[user.ProviderKey].Name,
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
			user.ProviderKey,
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

func (s AuthService) Logout(
	logger *zap.Logger,
	claims models.UserClaims,
) error {
	if claims.SID == "" {
		return apierrors.New(http.StatusUnauthorized, apierrors.CodeSessionRevoked)
	}
	if err := cache.RevokeSession(s.Cache, claims.UserID.String(), claims.SID); err != nil {
		logger.Error("Failed to revoke session", zap.Error(err))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	return nil
}
