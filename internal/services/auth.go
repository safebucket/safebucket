package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api/internal/activity"
	"api/internal/configuration"
	customerr "api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
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
)

type AuthService struct {
	DB             *gorm.DB
	JWTSecret      string
	Providers      configuration.Providers
	WebURL         string
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
}

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(m.Validate[models.AuthLoginBody]).Post("/login", handlers.CreateHandler(s.Login))
	r.With(m.Validate[models.AuthVerifyBody]).Post("/verify", handlers.CreateHandler(s.Verify))
	r.With(m.Validate[models.AuthRefreshBody]).Post("/refresh", handlers.CreateHandler(s.Refresh))

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
			r.Get("/callback", handlers.OpenIDCallbackHandler(s.WebURL, s.OpenIDCallback))
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
		return models.AuthLoginResponse{}, customerr.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, customerr.NewAPIError(403, "FORBIDDEN")
	}

	searchUser := models.User{
		Email:        body.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
	}
	result := s.DB.Where(searchUser, "email", "provider_type", "provider_key").Find(&searchUser)
	if result.RowsAffected == 1 {
		match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
		if err != nil || !match {
			return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
		}

		accessToken, err := h.NewAccessToken(
			s.JWTSecret,
			&searchUser,
			string(models.LocalProviderType),
		)
		if err != nil {
			return models.AuthLoginResponse{}, customerr.ErrGenerateAccessTokenFailed
		}

		refreshToken, err := h.NewRefreshToken(
			s.JWTSecret,
			&searchUser,
			string(models.LocalProviderType),
		)
		if err != nil {
			return models.AuthLoginResponse{}, customerr.ErrGenerateRefreshTokenFailed
		}

		return models.AuthLoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
	}
	return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
}

func (s AuthService) Verify(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthVerifyBody,
) (any, error) {
	data, err := h.ParseAccessToken(s.JWTSecret, body.AccessToken)
	return data, err
}

func (s AuthService) Refresh(
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.AuthRefreshBody,
) (models.AuthRefreshResponse, error) {
	refreshToken, err := h.ParseRefreshToken(s.JWTSecret, body.RefreshToken)
	if err != nil {
		return models.AuthRefreshResponse{}, err
	}
	accessToken, err := h.NewAccessToken(
		s.JWTSecret,
		&models.User{ID: refreshToken.UserID, Email: refreshToken.Email, Role: refreshToken.Role},
		refreshToken.Provider,
	)
	return models.AuthRefreshResponse{AccessToken: accessToken}, err
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
		return "", "", customerr.NewAPIError(403, "FORBIDDEN")
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
			return "", "", customerr.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
	}

	accessToken, err := h.NewAccessToken(s.JWTSecret, &searchUser, providerKey)
	if err != nil {
		return "", "", customerr.ErrGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(s.JWTSecret, &searchUser, providerKey)
	if err != nil {
		return "", "", customerr.ErrGenerateRefreshTokenFailed
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
	result := s.DB.Preload("User").
		Where("id = ? AND type = ?", challengeID, models.ChallengeTypePasswordReset).
		First(&challenge)

	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, customerr.NewAPIError(404, "CHALLENGE_NOT_FOUND")
	}

	if challenge.User == nil {
		logger.Error("Challenge has no associated user")
		return models.AuthLoginResponse{}, customerr.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return models.AuthLoginResponse{}, customerr.NewAPIError(410, "CHALLENGE_EXPIRED")
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
			s.DB.Delete(&challenge)
			return models.AuthLoginResponse{}, customerr.NewAPIError(403, "CHALLENGE_LOCKED")
		}

		if updateErr := s.DB.Model(&challenge).Update("failed_attempts", challenge.AttemptsLeft).Error; updateErr != nil {
			logger.Error("Failed to update attempts counter", zap.Error(updateErr))
		}

		return models.AuthLoginResponse{}, customerr.NewAPIError(401, "WRONG_CODE")
	}

	hashedPassword, err := h.CreateHash(body.NewPassword)
	if err != nil {
		logger.Error("Failed to hash new password", zap.Error(err))
		return models.AuthLoginResponse{}, customerr.NewAPIError(500, "PASSWORD_UPDATE_FAILED")
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(challenge.User).Update("hashed_password", hashedPassword)
		if updateResult.Error != nil {
			logger.Error("Failed to update password", zap.Error(updateResult.Error))
			tx.Rollback()
			return customerr.NewAPIError(500, "PASSWORD_UPDATE_FAILED")
		}

		deleteResult := tx.Delete(&challenge)
		if deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			tx.Rollback()
			return customerr.NewAPIError(500, "CHALLENGE_CLEANUP_FAILED")
		}

		return nil
	})
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	resetDate := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	successEvent := events.NewPasswordResetSuccess(
		s.Publisher,
		challenge.User.Email,
		s.WebURL,
		resetDate,
	)
	successEvent.Trigger()

	accessToken, err := h.NewAccessToken(
		s.JWTSecret,
		challenge.User,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.AuthLoginResponse{}, customerr.NewAPIError(
			500,
			"GENERATE_ACCESS_TOKEN_FAILED",
		)
	}

	refreshToken, err := h.NewRefreshToken(
		s.JWTSecret,
		challenge.User,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, customerr.NewAPIError(
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
	_ *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.PasswordResetRequestBody,
) (interface{}, error) {
	var user models.User
	result := s.DB.Where("email = ? AND provider_type = ?", body.Email, models.LocalProviderType).
		First(&user)

	if result.RowsAffected == 0 {
		return nil, nil
	}

	secret, err := h.GenerateSecret()
	if err != nil {
		return nil, customerr.NewAPIError(500, "PASSWORD_RESET_CREATION_FAILED")
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		return nil, customerr.NewAPIError(500, "PASSWORD_RESET_CREATION_FAILED")
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
		return nil, customerr.NewAPIError(500, "PASSWORD_RESET_CREATION_FAILED")
	}

	// Send password reset email
	event := events.NewPasswordResetChallenge(
		s.Publisher,
		secret,
		user.Email,
		challenge.ID.String(),
		s.WebURL,
	)
	event.Trigger()

	return nil, nil
}
