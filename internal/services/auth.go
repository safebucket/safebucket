package services

import (
	"api/internal/configuration"
	customerr "api/internal/errors"
	"api/internal/handlers"
	h "api/internal/helpers"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac/roles"
	"api/internal/sql"
	"context"
	"errors"
	"fmt"

	"github.com/alexedwards/argon2id"
	"github.com/casbin/casbin/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type AuthService struct {
	DB        *gorm.DB
	Enforcer  *casbin.Enforcer
	JWTSecret string
	Providers configuration.Providers
	WebUrl    string
}

func (s AuthService) Routes() chi.Router {
	r := chi.NewRouter()
	r.With(m.Validate[models.AuthLogin]).Post("/login", handlers.CreateHandler(s.Login))
	// TODO: MFA / ResetPassword / ResetTokens / IsAdmin (:= get user)
	r.With(m.Validate[models.AuthVerify]).Post("/verify", handlers.CreateHandler(s.Verify))
	r.With(m.Validate[models.AuthRefresh]).Post("/refresh", handlers.CreateHandler(s.Refresh))

	r.Route("/providers", func(r chi.Router) {
		r.Get("/", handlers.GetListHandler(s.GetProviderList))
		r.Route("/{provider}", func(r chi.Router) {
			r.Get("/begin", handlers.OpenIDBeginHandler(s.OpenIDBegin))
			r.Get("/callback", handlers.OpenIDCallbackHandler(s.WebUrl, s.OpenIDCallback))
		})
	})
	return r
}

func (s AuthService) Login(logger *zap.Logger, _ models.UserClaims, _ uuid.UUIDs, body models.AuthLogin) (models.AuthLoginResponse, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return models.AuthLoginResponse{}, customerr.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, customerr.NewAPIError(403, "FORBIDDEN")
	}

	searchUser := models.User{Email: body.Email, ProviderType: models.LocalProviderType, ProviderKey: string(models.LocalProviderType)}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 1 {
		match, err := argon2id.ComparePasswordAndHash(body.Password, searchUser.HashedPassword)
		if err != nil || !match {
			return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
		}

		accessToken, err := h.NewAccessToken(s.JWTSecret, &searchUser, string(models.LocalProviderType))
		if err != nil {
			return models.AuthLoginResponse{}, customerr.ErrorGenerateAccessTokenFailed
		}

		refreshToken, err := h.NewRefreshToken(s.JWTSecret, &searchUser, string(models.LocalProviderType))
		if err != nil {
			return models.AuthLoginResponse{}, customerr.ErrorGenerateRefreshTokenFailed
		}

		return models.AuthLoginResponse{AccessToken: accessToken, RefreshToken: refreshToken}, nil
	}
	return models.AuthLoginResponse{}, errors.New("invalid email / password combination")
}

func (s AuthService) Verify(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs, body models.AuthVerify) (any, error) {
	data, err := h.ParseAccessToken(s.JWTSecret, body.AccessToken)
	return data, err
}

func (s AuthService) Refresh(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs, body models.AuthRefresh) (models.AuthRefreshResponse, error) {
	refreshToken, err := h.ParseRefreshToken(s.JWTSecret, body.RefreshToken)
	if err != nil {
		return models.AuthRefreshResponse{}, err
	}
	accessToken, err := h.NewAccessToken(
		s.JWTSecret, &models.User{ID: refreshToken.UserID, Email: refreshToken.Email}, refreshToken.Provider,
	)
	return models.AuthRefreshResponse{AccessToken: accessToken}, err
}

func (s AuthService) GetProviderList(_ *zap.Logger, _ models.UserClaims, _ uuid.UUIDs) []models.ProviderResponse {
	var providers = make([]models.ProviderResponse, len(s.Providers))
	for id, provider := range s.Providers {
		if len(provider.Domains) == 0 {
			provider.Domains = []string{}
		}

		providers[provider.Order] = models.ProviderResponse{
			Id:      id,
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
		return "", fmt.Errorf("provider not found")
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, logger *zap.Logger, providerKey string, code string, nonce string,
) (string, string, error) {
	provider, ok := s.Providers[providerKey]
	if !ok {
		return "", "", fmt.Errorf("provider not found")
	}

	oauth2Token, err := provider.OauthConfig.Exchange(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("failed to exchange token %s", err.Error())
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", "", fmt.Errorf("no id_token field in oauth2 token")
	}

	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify ID token %s", err.Error())
	}

	if idToken.Nonce != nonce {
		return "", "", fmt.Errorf("nonce does not match")
	}

	userInfo, err := provider.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		return "", "", fmt.Errorf("failed to get user info %s", err.Error())
	}

	if !h.IsDomainAllowed(userInfo.Email, s.Providers[providerKey].Domains) {
		logger.Debug("Domain not allowed")
		return "", "", customerr.NewAPIError(403, "FORBIDDEN")
	}

	searchUser := models.User{Email: userInfo.Email, ProviderType: models.OIDCProviderType, ProviderKey: providerKey}
	result := s.DB.Where("email = ?", searchUser.Email).First(&searchUser)
	if result.RowsAffected == 0 {
		searchUser.ProviderType = models.OIDCProviderType
		searchUser.ProviderKey = providerKey

		err := sql.CreateUserWithRoleAndInvites(logger, s.DB, s.Enforcer, &searchUser, roles.AddUserToRoleUser)
		if err != nil {
			return "", "", customerr.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
	}

	accessToken, err := h.NewAccessToken(s.JWTSecret, &searchUser, providerKey)
	if err != nil {
		return "", "", customerr.ErrorGenerateAccessTokenFailed
	}

	refreshToken, err := h.NewRefreshToken(s.JWTSecret, &searchUser, providerKey)
	if err != nil {
		return "", "", customerr.ErrorGenerateRefreshTokenFailed
	}

	return accessToken, refreshToken, nil
}
