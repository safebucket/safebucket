package services

import (
	"context"
	"net/http"
	"strings"
	"unicode"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
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
		logger.Debug("Domain not allowed")
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

	firstName, lastName := oidcNames(userInfo)
	if !found {
		searchUser = models.User{
			FirstName:    firstName,
			LastName:     lastName,
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
	} else if namesChanged(&searchUser, firstName, lastName) {
		names := models.User{FirstName: firstName, LastName: lastName}
		if updateErr := s.DB.Model(&searchUser).Updates(names).Error; updateErr != nil {
			logger.Warn("Failed to set OIDC user names",
				zap.String("user_id", searchUser.ID.String()), zap.Error(updateErr))
		}
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

func oidcNames(userInfo *oidc.UserInfo) (string, string) {
	var claims struct {
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return "", ""
	}

	given := sanitizeName(claims.GivenName)
	family := sanitizeName(claims.FamilyName)
	full := sanitizeName(claims.Name)

	if family != "" {
		return given, family
	}
	if given != "" && given != full {
		return given, ""
	}

	lastSpace := strings.LastIndex(full, " ")
	if lastSpace < 0 {
		return full, ""
	}
	return full[:lastSpace], full[lastSpace+1:]
}

func namesChanged(user *models.User, firstName string, lastName string) bool {
	firstChanged := firstName != "" && firstName != user.FirstName
	lastChanged := lastName != "" && lastName != user.LastName
	return firstChanged || lastChanged
}

func sanitizeName(name string) string {
	const maxNameLength = 100

	cleaned := strings.Map(func(r rune) rune {
		if r == '<' || r == '>' || (!unicode.IsPrint(r) && !unicode.IsSpace(r)) {
			return -1
		}
		return r
	}, name)

	normalized := []rune(strings.Join(strings.Fields(cleaned), " "))
	if len(normalized) > maxNameLength {
		normalized = normalized[:maxNameLength]
	}
	return strings.TrimSpace(string(normalized))
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
