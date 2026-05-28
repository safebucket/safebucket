package services

import (
	"github.com/safebucket/safebucket/internal/auth/ldap"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
	"go.uber.org/zap"
)

func (s MFAService) verifyProviderPassword(
	logger *zap.Logger,
	user *models.User,
	password string,
) error {
	needsPassword := user.ProviderType == models.LocalProviderType ||
		user.ProviderType == models.LDAPProviderType
	if needsPassword && password == "" {
		return apierrors.NewAPIError(400, apierrors.CodeBadRequest)
	}

	switch user.ProviderType {
	case models.LocalProviderType:
		match, err := argon2id.ComparePasswordAndHash(password, user.HashedPassword)
		if err != nil {
			logger.Error("Failed to verify password", zap.Error(err))
			return apierrors.ErrInternalServer
		}
		if !match {
			return apierrors.NewAPIError(401, apierrors.CodeInvalidPassword)
		}
		return nil
	case models.LDAPProviderType:
		provider, ok := s.Providers[user.ProviderKey]
		if !ok || provider.LDAP == nil {
			logger.Error("LDAP provider missing or unconfigured for MFA re-auth",
				zap.String("provider_key", user.ProviderKey))
			return apierrors.ErrInternalServer
		}
		if _, err := ldap.AuthenticateAndFetch(*provider.LDAP, user.Email, password); err != nil {
			return mapLDAPAuthError(logger, user.ProviderKey, err, apierrors.CodeInvalidPassword)
		}
		return nil
	case models.OIDCProviderType: //TODO: implement re-login here
		return nil
	default:
		return nil
	}
}
