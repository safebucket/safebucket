package services

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/auth/rolemapping"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/mfa"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/outbox"
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
		return "", apierrors.New(http.StatusNotFound, apierrors.CodeProviderNotFound)
	}

	url := provider.OauthConfig.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.AccessTypeOffline)
	return url, nil
}

func (s AuthService) OpenIDCallback(
	ctx context.Context, logger *zap.Logger, providerKey string, code string, nonce string,
) (models.OIDCCallbackResult, error) {
	provider, ok := s.Providers[providerKey]
	if !ok || provider.Type != models.OIDCProviderType {
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusNotFound, apierrors.CodeProviderNotFound)
	}

	oauth2Token, err := provider.OauthConfig.Exchange(ctx, code)
	if err != nil {
		logger.Error("Failed to exchange OAuth2 token", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusBadRequest, apierrors.CodeOAuthExchangeFailed)
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusBadRequest, apierrors.CodeIDTokenMissing)
	}

	idToken, err := provider.Verifier.Verify(ctx, rawIDToken)
	if err != nil {
		logger.Error("Failed to verify ID token", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusUnauthorized, apierrors.CodeIDTokenVerifyFailed)
	}

	if idToken.Nonce != nonce {
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusBadRequest, apierrors.CodeOIDCNonceMismatch)
	}

	userInfo, err := provider.Provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		logger.Error("Failed to get user info from provider", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusBadGateway, apierrors.CodeOAuthUserinfoFailed)
	}

	if !h.IsDomainAllowed(userInfo.Email, provider.Domains) {
		if strings.TrimSpace(userInfo.Email) == "" {
			logger.Warn("OIDC user has no email, login rejected",
				zap.String("provider", providerKey),
				zap.String("subject", userInfo.Subject))
		} else {
			logger.Debug("Domain not allowed")
		}
		return models.OIDCCallbackResult{}, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	searchUser, found, err := sql.FindUserByIdentityProvider(
		s.DB, userInfo.Email, models.OIDCProviderType, providerKey, false,
	)
	if err != nil {
		logger.Error("Failed to look up OIDC user", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}
	if !found {
		searchUser = models.User{
			Email:        userInfo.Email,
			ProviderType: models.OIDCProviderType,
			ProviderKey:  providerKey,
			Role:         models.RoleUser,
		}
		if createErr := sql.CreateUserWithInvites(logger, s.DB, &searchUser); createErr != nil {
			logger.Error("Failed to create OIDC user", zap.Error(createErr))
			return models.OIDCCallbackResult{}, apierrors.New(
				http.StatusInternalServerError,
				apierrors.CodeInternalServerError,
			)
		}
	}

	if syncErr := s.applyIDPRoleMapping(logger, provider, idToken, userInfo, &searchUser); syncErr != nil {
		return models.OIDCCallbackResult{}, syncErr
	}

	verifiedCount, countErr := sql.CountVerifiedMFADevices(s.DB, searchUser.ID)
	if countErr != nil {
		logger.Error("Failed to count verified MFA devices", zap.Error(countErr))
		return models.OIDCCallbackResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	if provider.MFARequired || verifiedCount > 0 {
		mfaToken, mfaErr := mfa.HandleMFARequired(logger, s.AuthConfig, &searchUser)
		if mfaErr != nil {
			return models.OIDCCallbackResult{}, mfaErr
		}
		return models.OIDCCallbackResult{MFAToken: mfaToken, MFARequired: true}, nil
	}

	return s.issueOIDCSession(logger, &searchUser, providerKey)
}

func (s AuthService) issueOIDCSession(
	logger *zap.Logger, user *models.User, providerKey string,
) (models.OIDCCallbackResult, error) {
	sid := uuid.New().String()
	if sessionErr := cache.CreateSession(s.Cache, user.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return models.OIDCCallbackResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	accessToken, err := h.NewAccessToken(s.AuthConfig.TokenSecret, user, providerKey, sid)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	refreshToken, err := h.NewRefreshToken(s.AuthConfig.TokenSecret, user, providerKey, sid)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.OIDCCallbackResult{}, apierrors.New(
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
			ProviderType: string(models.OIDCProviderType),
			ProviderName: s.Providers[providerKey].Name,
		}),
	}
	if logErr := outbox.AddActivities(s.DB, action); logErr != nil {
		logger.Error("Failed to log login activity", zap.Error(logErr))
	}

	return models.OIDCCallbackResult{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s AuthService) applyIDPRoleMapping(
	logger *zap.Logger,
	provider configuration.Provider,
	idToken *oidc.IDToken,
	userInfo *oidc.UserInfo,
	user *models.User,
) error {
	if provider.RoleMapping == nil {
		return nil
	}

	idTokenClaims := map[string]json.RawMessage{}
	if claimsErr := idToken.Claims(&idTokenClaims); claimsErr != nil {
		logger.Error("Failed to decode ID token claims",
			zap.String("provider", user.ProviderKey),
			zap.String("user_id", user.ID.String()),
			zap.Error(claimsErr))
		idTokenClaims = nil
	}

	userInfoClaims := map[string]json.RawMessage{}
	if claimsErr := userInfo.Claims(&userInfoClaims); claimsErr != nil {
		logger.Error("Failed to decode UserInfo claims",
			zap.String("provider", user.ProviderKey),
			zap.String("user_id", user.ID.String()),
			zap.Error(claimsErr))
		userInfoClaims = nil
	}

	groups, found, err := rolemapping.OIDCGroups(idTokenClaims, userInfoClaims, provider.GroupsClaim)
	if err != nil {
		logger.Error("Failed to decode groups claim",
			zap.String("provider", user.ProviderKey),
			zap.String("user_id", user.ID.String()),
			zap.String("claim", provider.GroupsClaim),
			zap.Error(err))
	} else if !found {
		logger.Warn("Groups claim missing from IdP response",
			zap.String("provider", user.ProviderKey),
			zap.String("user_id", user.ID.String()),
			zap.String("claim", provider.GroupsClaim))
	}

	return s.syncIDPRole(logger, user, provider, groups)
}
