package services

import (
	"errors"
	"net/http"
	"strings"

	"github.com/safebucket/safebucket/internal/activity"
	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/cache"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/mfa"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"go.uber.org/zap"
)

func (s AuthService) LDAPLogin(
	isSecure bool,
	logger *zap.Logger,
	providerKey string,
	body models.AuthLoginBody,
) (handlers.AuthFlowResult, error) {
	provider, ok := s.Providers[providerKey]
	if !ok || provider.Type != models.LDAPProviderType {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusNotFound, apierrors.CodeProviderNotFound)
	}

	if !h.IsDomainAllowed(body.Email, provider.Domains) {
		logger.Debug("Domain not allowed for LDAP provider", zap.String("provider", providerKey))
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	ldapUser, err := s.authenticateLDAP(logger, providerKey, provider.LDAPConfig, body.Email, body.Password)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	user, err := s.findOrCreateLDAPUser(logger, providerKey, ldapUser)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	verifiedDevices := user.GetVerifiedDevices()
	hasMFA := len(verifiedDevices) > 0

	if hasMFA || s.AuthConfig.MFARequired {
		restrictedToken, mfaErr := mfa.HandleMFARequired(logger, s.AuthConfig, &user)
		if mfaErr != nil {
			return handlers.AuthFlowResult{}, mfaErr
		}
		return handlers.AuthFlowResult{
			Status:  http.StatusOK,
			Body:    models.AuthLoginResponse{MFARequired: true},
			Cookies: handlers.BuildMFACookie(isSecure, restrictedToken),
		}, nil
	}

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &user)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	if err = cache.CreateSession(s.Cache, user.ID.String(), sid); err != nil {
		logger.Error("Failed to create LDAP session", zap.Error(err))
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
			ProviderType: string(models.LDAPProviderType),
			ProviderName: provider.Name,
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log LDAP login activity", zap.Error(logErr))
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

func (s AuthService) authenticateLDAP(
	logger *zap.Logger,
	providerKey string,
	cfg *ldapclient.Config,
	email, password string,
) (ldapclient.User, error) {
	ldapUser, err := ldapclient.AuthenticateAndFetch(*cfg, email, password)
	if err != nil {
		return ldapclient.User{}, mapLDAPAuthError(logger, providerKey, err)
	}
	return ldapUser, nil
}

func (s AuthService) findOrCreateLDAPUser(
	logger *zap.Logger,
	providerKey string,
	ldapUser ldapclient.User,
) (models.User, error) {
	email := normalizeExternalEmail(ldapUser.Email)

	user, found, err := sql.FindUserByIdentityProvider(
		s.DB, email, models.LDAPProviderType, providerKey, true,
	)
	if err != nil {
		logger.Error("Failed to look up LDAP user", zap.Error(err))
		return models.User{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	if found {
		return user, nil
	}

	user = models.User{
		Email:        email,
		ProviderType: models.LDAPProviderType,
		ProviderKey:  providerKey,
		Role:         models.RoleUser,
	}
	if createErr := sql.CreateUserWithInvites(logger, s.DB, &user); createErr != nil {
		logger.Error("Failed to create LDAP user", zap.Error(createErr))
		return models.User{}, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
	return user, nil
}

func mapLDAPAuthError(logger *zap.Logger, providerKey string, err error) error {
	if errors.Is(err, ldapclient.ErrInvalidCredentials) {
		return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidCredentials)
	}
	logger.Error("LDAP provider error",
		zap.String("provider", providerKey),
		zap.Error(err),
	)
	return apierrors.New(http.StatusServiceUnavailable, apierrors.CodeAuthProviderUnavailable)
}

func normalizeExternalEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
