package services

import (
	"errors"

	"github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s AuthService) LDAPLogin(
	isSecure bool,
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	providerKey string,
	body models.AuthLoginBody,
) (handlers.AuthFlowResult, error) {
	provider, ok := s.Providers[providerKey]
	if !ok || provider.Type != models.LDAPProviderType {
		return handlers.AuthFlowResult{}, apierrors.NewAPIError(404, apierrors.CodeProviderNotFound)
	}

	user, err := s.authenticateLDAP(logger, providerKey, provider, body)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	return s.issueLoginResult(isSecure, logger, &user, providerKey, provider)
}

func (s AuthService) authenticateLDAP(
	logger *zap.Logger,
	providerKey string,
	provider configuration.Provider,
	body models.AuthLoginBody,
) (models.User, error) {
	if provider.LDAP == nil {
		logger.Error("LDAP provider missing client config", zap.String("provider", providerKey))
		return models.User{}, apierrors.ErrInternalServer
	}

	ldapUser, err := ldap.AuthenticateAndFetch(*provider.LDAP, body.Email, body.Password)
	if err != nil {
		return models.User{}, mapLDAPAuthError(logger, providerKey, err, apierrors.CodeInvalidCredentials)
	}

	return s.findOrCreateLDAPUser(logger, providerKey, ldapUser)
}

func (s AuthService) findOrCreateLDAPUser(
	logger *zap.Logger,
	providerKey string,
	ldapUser *ldap.User,
) (models.User, error) {
	email := normalizeExternalEmail(ldapUser.Email)

	user, found, err := sql.FindUserByProviderIdentity(
		s.DB, email, models.LDAPProviderType, providerKey, true,
	)
	if err != nil {
		logger.Error("Failed to look up LDAP user", zap.Error(err))
		return models.User{}, apierrors.ErrInternalServer
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
	if _, err = sql.CreateOrGetUser(logger, s.DB, &user); err != nil {
		logger.Error("Failed to create LDAP user", zap.Error(err))
		return models.User{}, apierrors.ErrInternalServer
	}
	if err = s.DB.Preload("MFADevices", "is_verified = ?", true).
		First(&user, user.ID).Error; err != nil {
		logger.Error("Failed to load LDAP user after create",
			zap.String("email", email), zap.Error(err))
		return models.User{}, apierrors.ErrInternalServer
	}

	return user, nil
}

func mapLDAPAuthError(logger *zap.Logger, providerKey string, err error, missCode string) error {
	switch {
	case errors.Is(err, ldap.ErrInvalidCredentials),
		errors.Is(err, ldap.ErrUserNotFound):
		return apierrors.NewAPIError(401, missCode)
	case errors.Is(err, ldap.ErrDirectoryUnavailable),
		errors.Is(err, ldap.ErrServiceBindFailed):
		logger.Error("LDAP directory unavailable",
			zap.String("provider", providerKey),
			zap.Error(err))
		return apierrors.NewAPIError(503, apierrors.CodeAuthProviderUnavailable)
	case errors.Is(err, ldap.ErrMissingEmail):
		logger.Error("LDAP user authenticated but the directory entry is missing the email attribute",
			zap.String("provider", providerKey),
			zap.Error(err))
		return apierrors.NewAPIError(503, apierrors.CodeAuthProviderUnavailable)
	default:
		logger.Error("LDAP authentication failed unexpectedly",
			zap.String("provider", providerKey),
			zap.Error(err))
		return apierrors.NewAPIError(503, apierrors.CodeAuthProviderUnavailable)
	}
}
