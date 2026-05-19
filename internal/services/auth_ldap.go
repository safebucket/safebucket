package services

import (
	"errors"
	"strings"

	"github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/sql"

	"go.uber.org/zap"
)

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
	firstName, lastName := splitDisplayName(ldapUser.DisplayName)

	user, found, err := sql.FindUserByProviderIdentity(
		s.DB, email, models.LDAPProviderType, providerKey, true,
	)
	if err != nil {
		logger.Error("Failed to look up LDAP user", zap.Error(err))
		return models.User{}, apierrors.ErrInternalServer
	}

	if !found {
		user = models.User{
			FirstName:    firstName,
			LastName:     lastName,
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
	}

	sql.SyncUserAttributes(logger, s.DB, &user, firstName, lastName)
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
	default:
		logger.Error("LDAP authentication failed",
			zap.String("provider", providerKey),
			zap.Error(err))
		return apierrors.NewAPIError(401, missCode)
	}
}

func splitDisplayName(display string) (string, string) {
	display = strings.TrimSpace(display)
	if display == "" {
		return "", ""
	}
	first, last, ok := strings.Cut(display, " ")
	if !ok {
		return first, ""
	}
	return first, strings.TrimSpace(last)
}
