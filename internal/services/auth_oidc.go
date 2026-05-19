package services

import (
	"context"
	"errors"
	"fmt"

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
		return "", apierrors.NewAPIError(404, apierrors.CodeProviderNotFound)
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, logger *zap.Logger, providerKey string, code string, nonce string,
) (string, string, error) {
	provider, ok := s.Providers[providerKey]
	if !ok || provider.Type != models.OIDCProviderType {
		return "", "", apierrors.NewAPIError(404, apierrors.CodeProviderNotFound)
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

	email := normalizeExternalEmail(userInfo.Email)
	var oidcClaims struct {
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	_ = userInfo.Claims(&oidcClaims)

	searchUser, found, err := sql.FindUserByProviderIdentity(
		s.DB, email, models.OIDCProviderType, providerKey, false,
	)
	if err != nil {
		logger.Error("Failed to look up OIDC user", zap.Error(err))
		return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}
	if found {
		sql.SyncUserAttributes(logger, s.DB, &searchUser, oidcClaims.GivenName, oidcClaims.FamilyName)
	} else {
		searchUser = models.User{
			Email:        email,
			FirstName:    oidcClaims.GivenName,
			LastName:     oidcClaims.FamilyName,
			ProviderType: models.OIDCProviderType,
			ProviderKey:  providerKey,
			Role:         models.RoleUser,
		}
		created, createErr := sql.CreateOrGetUser(logger, s.DB, &searchUser)
		if createErr != nil {
			logger.Error("Failed to create OIDC user", zap.Error(createErr))
			return "", "", apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
		if !created {
			sql.SyncUserAttributes(logger, s.DB, &searchUser, oidcClaims.GivenName, oidcClaims.FamilyName)
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
