package services

import (
	"errors"
	"net/http"
	"strings"

	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
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

	ldapUser, err := ldapclient.AuthenticateAndFetch(*provider.LDAPConfig, body.Email, body.Password)
	if err != nil {
		return handlers.AuthFlowResult{}, mapLDAPAuthError(logger, providerKey, err)
	}

	email := normalizeExternalEmail(ldapUser.Email)

	user, found, err := sql.FindUserByIdentityProvider(
		s.DB, email, models.LDAPProviderType, providerKey, true,
	)
	if err != nil {
		logger.Error("Failed to look up LDAP user", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	if !found {
		user = models.User{
			Email:        email,
			ProviderType: models.LDAPProviderType,
			ProviderKey:  providerKey,
			Role:         models.RoleUser,
		}
		if createErr := sql.CreateUserWithInvites(logger, s.DB, &user); createErr != nil {
			logger.Error("Failed to create LDAP user", zap.Error(createErr))
			return handlers.AuthFlowResult{}, apierrors.New(
				http.StatusInternalServerError,
				apierrors.CodeInternalServerError,
			)
		}
	}

	return s.finalizeLogin(isSecure, logger, &user, models.LDAPProviderType, provider.Name, provider.MFARequired)
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
