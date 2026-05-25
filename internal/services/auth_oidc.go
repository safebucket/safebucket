package services

import (
	"context"
	"net/http"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func (s AuthService) OpenIDBegin(providerName string, state string, nonce string) (string, error) {
	provider, ok := s.Providers[providerName]
	if !ok || provider.Type != models.OIDCProviderType {
		return "", apierrors.NewAPIError(http.StatusNotFound, apierrors.CodeProviderNotFound)
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, logger *zap.Logger, providerKey string, code string, nonce string,
) (string, string, error) {
	provider, ok := s.Providers[providerKey]
	if !ok || provider.Type != models.OIDCProviderType {
		return "", "", apierrors.NewAPIError(http.StatusNotFound, apierrors.CodeProviderNotFound)
	}

	oauth2Token, err := provider.OauthConfig.Exchange(ctx, code)
	if err != nil {
		logger.Error("Failed to exchange OAuth2 token", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusBadRequest, apierrors.CodeOAuthExchangeFailed)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return "", "", apierrors.NewAPIError(http.StatusBadRequest, apierrors.CodeIDTokenMissing)
	}

	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		logger.Error("Failed to verify ID token", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusUnauthorized, apierrors.CodeIDTokenVerifyFailed)
	}

	if idToken.Nonce != nonce {
		return "", "", apierrors.NewAPIError(http.StatusBadRequest, apierrors.CodeOIDCNonceMismatch)
	}

	userInfo, err := provider.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		logger.Error("Failed to get user info from provider", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusBadGateway, apierrors.CodeOAuthUserinfoFailed)
	}

	if !h.IsDomainAllowed(userInfo.Email, s.Providers[providerKey].Domains) {
		logger.Debug("Domain not allowed")
		return "", "", apierrors.NewAPIError(http.StatusForbidden, apierrors.CodeForbidden)
	}

	email := normalizeExternalEmail(userInfo.Email)

	searchUser, found, err := sql.FindUserByProviderIdentity(
		s.DB, email, models.OIDCProviderType, providerKey, false,
	)
	if err != nil {
		logger.Error("Failed to look up OIDC user", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	if !found {
		searchUser = models.User{
			Email:        email,
			ProviderType: models.OIDCProviderType,
			ProviderKey:  providerKey,
			Role:         models.RoleUser,
		}
		if _, createErr := sql.CreateOrGetUser(logger, s.DB, &searchUser); createErr != nil {
			logger.Error("Failed to create OIDC user", zap.Error(createErr))
			return "", "", apierrors.NewAPIError(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
	}

	sid := uuid.New().String()
	if sessionErr := cache.CreateSession(s.Cache, searchUser.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return "", "", apierrors.NewAPIError(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.TokenSecret,
		&searchUser,
		providerKey,
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.TokenSecret,
		&searchUser,
		providerKey,
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return "", "", apierrors.NewAPIError(http.StatusInternalServerError, apierrors.CodeInternalServerError)
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
