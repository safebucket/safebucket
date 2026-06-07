package services

import (
	"errors"
	"net/http"
	"strings"

	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/alexedwards/argon2id"
	"go.uber.org/zap"
)

func (s MFAService) verifyMFAStepUp(logger *zap.Logger, user *models.User, password, code string) error {
	if user.ProviderType == models.OIDCProviderType {
		return s.verifyTOTPStepUp(logger, user, code)
	}
	return s.verifyProviderPassword(logger, user, password)
}

func (s MFAService) verifyTOTPStepUp(logger *zap.Logger, user *models.User, code string) error {
	if strings.TrimSpace(code) == "" {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
	}

	var devices []models.MFADevice
	s.DB.Where("user_id = ? AND is_verified = ?", user.ID, true).Find(&devices)
	if len(devices) == 0 {
		return apierrors.New(http.StatusBadRequest, apierrors.CodeMFANotEnabled)
	}

	attempts, err := cache.GetMFAAttempts(s.Cache, user.ID.String())
	if err != nil {
		logger.Error("Rate limit check failed - denying request", zap.Error(err))
		return apierrors.New(http.StatusServiceUnavailable, apierrors.CodeServiceUnavailable)
	}
	if attempts >= configuration.MFAMaxAttempts {
		logger.Warn("MFA step-up rate limited", zap.String("user_id", user.ID.String()))
		return apierrors.New(http.StatusTooManyRequests, apierrors.CodeMFARateLimited)
	}

	for i := range devices {
		secret, decErr := h.DecryptSecret(devices[i].EncryptedSecret, []byte(s.AuthConfig.MFAEncryptionKey))
		if decErr != nil {
			logger.Error("Failed to decrypt TOTP secret for step-up", zap.Error(decErr))
			continue
		}
		if !h.ValidateTOTPCode(secret, code) {
			continue
		}

		unused, usedErr := cache.MarkTOTPCodeUsed(s.Cache, devices[i].ID.String(), code)
		if usedErr != nil {
			logger.Error("Failed to mark TOTP code as used", zap.Error(usedErr))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		if !unused {
			logger.Warn("TOTP code replay attempt detected during step-up",
				zap.String("device_id", devices[i].ID.String()))
			return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
		}

		if resetErr := cache.ResetMFAAttempts(s.Cache, user.ID.String()); resetErr != nil {
			logger.Error("Failed to reset MFA attempts", zap.Error(resetErr))
		}
		return nil
	}

	if incErr := cache.IncrementMFAAttempts(s.Cache, user.ID.String()); incErr != nil {
		logger.Error("Failed to increment MFA attempts", zap.Error(incErr))
	}
	logger.Warn("Invalid TOTP code for MFA step-up", zap.String("user_id", user.ID.String()))
	return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
}

func (s MFAService) verifyAddDeviceStepUp(logger *zap.Logger, user *models.User, password, code string) error {
	if user.ProviderType == models.OIDCProviderType {
		var verifiedCount int64
		if err := s.DB.Model(&models.MFADevice{}).
			Where("user_id = ? AND is_verified = ?", user.ID, true).
			Count(&verifiedCount).Error; err != nil {
			logger.Error("Failed to count verified MFA devices for step-up", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		if verifiedCount == 0 {
			return nil
		}
		return s.verifyTOTPStepUp(logger, user, code)
	}
	return s.verifyProviderPassword(logger, user, password)
}

func (s MFAService) verifyProviderPassword(logger *zap.Logger, user *models.User, password string) error {
	switch user.ProviderType {
	case models.LocalProviderType:
		if password == "" {
			return apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
		}
		match, err := argon2id.ComparePasswordAndHash(password, user.HashedPassword)
		if err != nil {
			logger.Error("Failed to compare password and hash", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		if !match {
			logger.Warn("Invalid password provided for MFA operation", zap.String("user_id", user.ID.String()))
			return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidPassword)
		}
		return nil

	case models.LDAPProviderType:
		if strings.TrimSpace(password) == "" {
			return apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
		}
		provider, ok := s.Providers[user.ProviderKey]
		if !ok || provider.Type != models.LDAPProviderType || provider.LDAPConfig == nil {
			logger.Error("LDAP provider not found for MFA re-auth", zap.String("provider", user.ProviderKey))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}
		if _, err := ldapclient.AuthenticateAndFetch(*provider.LDAPConfig, user.Email, password); err != nil {
			if errors.Is(err, ldapclient.ErrInvalidCredentials) {
				logger.Warn("Invalid LDAP credentials for MFA operation", zap.String("user_id", user.ID.String()))
				return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidPassword)
			}
			logger.Error("LDAP re-auth failed for MFA operation",
				zap.String("provider", user.ProviderKey), zap.Error(err))
			return apierrors.New(http.StatusServiceUnavailable, apierrors.CodeAuthProviderUnavailable)
		}
		return nil

	case models.OIDCProviderType:
		logger.Error("verifyProviderPassword reached for OIDC user", zap.String("user_id", user.ID.String()))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)

	default:
		logger.Error("Unsupported provider type for password MFA re-auth",
			zap.String("provider_type", string(user.ProviderType)))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
}
