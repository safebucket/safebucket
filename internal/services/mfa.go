package services

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	"github.com/safebucket/safebucket/internal/mfa"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/notifier"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MFAService struct {
	DB             *gorm.DB
	Cache          cache.ICache
	AuthConfig     models.AuthConfig
	Providers      configuration.Providers
	Publisher      messaging.IPublisher
	Notifier       notifier.INotifier
	ActivityLogger activity.IActivityLogger
}

func (s MFAService) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/devices", func(r chi.Router) {
		r.Get("/", handlers.GetOneHandler(s.ListDevices))

		r.With(m.Validate[models.MFADeviceSetupBody]).
			Post("/", handlers.CreateHandler(s.AddDevice))

		r.Route("/{id0}", func(r chi.Router) {
			r.Get("/", handlers.GetOneHandler(s.GetDevice))

			r.With(m.Validate[models.MFADeviceUpdateBody]).
				Patch("/", handlers.BodyHandler(s.UpdateDevice))

			r.With(m.Validate[models.MFADeviceRemoveBody]).
				Delete("/", handlers.BodyHandler(s.RemoveDevice))

			r.With(m.Validate[models.MFADeviceVerifyBody]).
				Post("/verify", s.verifyDeviceHandler())
		})
	})
	return r
}

func (s MFAService) ListDevices(
	_ *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
) (models.MFADevicesListResponse, error) {
	userID := claims.UserID

	var devices []models.MFADevice
	result := s.DB.Where("user_id = ? AND is_verified = ?", userID, true).
		Order("is_default DESC, created_at ASC").
		Find(&devices)
	if result.Error != nil {
		return models.MFADevicesListResponse{}, result.Error
	}

	return models.MFADevicesListResponse{
		Devices: devices,
	}, nil
}

func (s MFAService) AddDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	_ uuid.UUIDs,
	body models.MFADeviceSetupBody,
) (models.MFADeviceSetupResponse, error) {
	userID := claims.UserID

	user, err := sql.GetUserByID(s.DB, userID)
	if err != nil {
		return models.MFADeviceSetupResponse{}, err
	}

	var count int64
	result := s.DB.Model(&models.MFADevice{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		logger.Error("Failed to count MFA devices", zap.Error(result.Error))
		return models.MFADeviceSetupResponse{}, result.Error
	}
	if count >= int64(configuration.MaxMFADevicesPerUser) {
		return models.MFADeviceSetupResponse{}, apierrors.New(http.StatusBadRequest, apierrors.CodeMaxMFADevicesReached)
	}

	isRestricted := claims.AudienceString() == configuration.AudienceMFALogin ||
		claims.AudienceString() == configuration.AudienceMFAReset

	if isRestricted {
		verifiedCount, countErr := sql.CountVerifiedMFADevices(s.DB, userID)
		if countErr != nil {
			logger.Error("Failed to count verified MFA devices", zap.Error(countErr))
			return models.MFADeviceSetupResponse{}, countErr
		}
		if verifiedCount > 0 {
			logger.Warn("restricted token used for non-initial device setup",
				zap.String("userID", claims.UserID.String()),
				zap.Int64("verifiedDeviceCount", verifiedCount))
			return models.MFADeviceSetupResponse{}, apierrors.New(
				http.StatusForbidden,
				apierrors.CodeMFASetupRestricted,
			)
		}
	} else {
		if err = s.verifyAddDeviceStepUp(logger, &user, body.Password, body.Code); err != nil {
			return models.MFADeviceSetupResponse{}, err
		}
	}

	var existing models.MFADevice
	result = s.DB.Where("user_id = ? AND name = ? AND is_verified = ?", userID, body.Name, true).Find(&existing)
	if result.RowsAffected > 0 {
		return models.MFADeviceSetupResponse{}, apierrors.New(http.StatusConflict, apierrors.CodeMFADeviceNameExists)
	}

	totpKey, err := h.GenerateTOTPSecret(user.Email)
	if err != nil {
		logger.Error("Failed to generate TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeMFASetupFailed,
		)
	}

	encryptedSecret, err := h.EncryptSecret(totpKey.Secret, []byte(s.AuthConfig.MFAEncryptionKey))
	if err != nil {
		logger.Error("Failed to encrypt TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeMFASetupFailed,
		)
	}

	device := models.MFADevice{
		UserID:          userID,
		Name:            body.Name,
		Type:            models.MFADeviceTypeTOTP,
		EncryptedSecret: encryptedSecret,
		IsDefault:       false,
		IsVerified:      false,
	}

	if err = s.DB.Create(&device).Error; err != nil {
		logger.Error("Failed to create MFA device", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeMFASetupFailed,
		)
	}

	action := models.Activity{
		Message: activity.MFADeviceEnrolled,
		Object:  device.ToActivity(),
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     activity.MFADeviceEnrolled,
			UserID:     userID.String(),
			ObjectType: rbac.ResourceMFADevice.String(),
			DeviceID:   device.ID.String(),
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log MFA device enrollment activity", zap.Error(logErr))
	}

	logger.Info("MFA device setup initiated",
		zap.String("user_id", userID.String()),
		zap.String("device_id", device.ID.String()),
		zap.String("device_name", body.Name))

	return models.MFADeviceSetupResponse{
		DeviceID:  device.ID,
		Secret:    totpKey.Secret,
		QRCodeURI: totpKey.URL,
		Issuer:    configuration.AppName,
	}, nil
}

func (s MFAService) GetDevice(
	_ *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
) (models.MFADevice, error) {
	userID := claims.UserID
	deviceID := ids[0]

	var device models.MFADevice
	result := s.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device)
	if result.RowsAffected == 0 {
		return models.MFADevice{}, apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
	}

	return device, nil
}

func (s MFAService) VerifyDevice(
	isSecure bool,
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceVerifyBody,
) (handlers.AuthFlowResult, error) {
	userID := claims.UserID
	deviceID := ids[0]

	var user models.User
	var deviceName string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := sql.GetUserByID(tx, userID); err != nil {
			return err
		}

		var device models.MFADevice
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			First(&device)
		if result.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
		}

		if device.IsVerified {
			return apierrors.New(http.StatusConflict, apierrors.CodeMFADeviceAlreadyVerified)
		}

		deviceName = device.Name

		secret, err := h.DecryptSecret(device.EncryptedSecret, []byte(s.AuthConfig.MFAEncryptionKey))
		if err != nil {
			logger.Error("Failed to decrypt TOTP secret", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeMFAVerificationFailed)
		}

		attempts, err := cache.GetMFAAttempts(s.Cache, userID.String())
		if err != nil {
			logger.Error("Rate limit check failed - denying request", zap.Error(err))
			return apierrors.New(http.StatusServiceUnavailable, apierrors.CodeServiceUnavailable)
		}
		if attempts >= configuration.MFAMaxAttempts {
			logger.Warn("MFA device verification rate limited",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.New(http.StatusTooManyRequests, apierrors.CodeMFARateLimited)
		}

		if !h.ValidateTOTPCode(secret, body.Code) {
			if incErr := cache.IncrementMFAAttempts(s.Cache, userID.String()); incErr != nil {
				logger.Error("Failed to increment MFA attempts", zap.Error(incErr))
			}

			logger.Warn("MFA device verification failed - invalid code",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
		}

		if err = cache.ResetMFAAttempts(s.Cache, userID.String()); err != nil {
			logger.Error("Failed to reset MFA attempts", zap.Error(err))
		}

		unused, err := cache.MarkTOTPCodeUsed(s.Cache, deviceID.String(), body.Code)
		if err != nil {
			logger.Error("Failed to mark TOTP code as used", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeMFAVerificationFailed)
		}
		if !unused {
			logger.Warn("TOTP code replay attempt detected",
				zap.String("device_id", deviceID.String()))
			return apierrors.New(http.StatusUnauthorized, apierrors.CodeInvalidMFACode)
		}

		var existingDefaultCount int64
		tx.Model(&models.MFADevice{}).
			Where("user_id = ? AND is_verified = ? AND is_default = ? AND id != ?",
				userID, true, true, deviceID).
			Count(&existingDefaultCount)

		shouldBeDefault := existingDefaultCount == 0

		now := time.Now()
		if err = tx.Model(&device).Updates(map[string]any{
			"is_verified":  true,
			"is_default":   shouldBeDefault,
			"verified_at":  now,
			"last_used_at": now,
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	action := models.Activity{
		Message: activity.MFADeviceVerified,
		Object: models.MFADeviceActivity{
			ID:   deviceID,
			Name: deviceName,
		},
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     activity.MFADeviceVerified,
			UserID:     userID.String(),
			ObjectType: rbac.ResourceMFADevice.String(),
			DeviceID:   deviceID.String(),
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log MFA device verification activity", zap.Error(logErr))
	}

	s.DB.Preload("MFADevices", "is_verified = ?", true).First(&user, userID)

	if claims.AudienceString() == configuration.AudienceMFAReset {
		var restrictedToken string
		restrictedToken, err = h.NewRestrictedAccessToken(
			s.AuthConfig.TokenSecret,
			&user,
			configuration.AudienceMFAReset,
			true,
			claims.ChallengeID,
		)
		if err != nil {
			logger.Error("Failed to generate restricted access token", zap.Error(err))
			return handlers.AuthFlowResult{}, apierrors.New(
				http.StatusInternalServerError,
				apierrors.CodeTokenGenerationFailed,
			)
		}

		logger.Info("MFA device verified (password reset flow)",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()))

		return handlers.AuthFlowResult{
			Status:  http.StatusOK,
			Body:    models.AuthLoginResponse{},
			Cookies: handlers.BuildMFACookie(isSecure, restrictedToken),
		}, nil
	}

	logger.Info("MFA device verified and enabled",
		zap.String("user_id", userID.String()),
		zap.String("device_id", deviceID.String()))

	go func() {
		if notifyErr := s.Notifier.NotifyFromTemplate(
			user.Email,
			"New MFA Device Enrolled - Safebucket",
			"mfa_device_enrolled",
			map[string]string{
				"DeviceName": deviceName,
				"WebURL":     s.AuthConfig.WebURL,
			},
		); notifyErr != nil {
			logger.Warn("Failed to send MFA device enrollment notification",
				zap.Error(notifyErr),
				zap.String("user_id", userID.String()),
				zap.String("email", user.Email))
		}
	}()

	sid, tokens, err := mfa.GenerateTokens(s.AuthConfig, &user)
	if err != nil {
		return handlers.AuthFlowResult{}, err
	}

	if sessionErr := cache.CreateSession(s.Cache, userID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
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

func (s MFAService) verifyDeviceHandler() http.HandlerFunc {
	return handlers.AuthFlowHandler(s.AuthConfig.CookieSecureForce, s.VerifyDevice)
}

func (s MFAService) UpdateDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceUpdateBody,
) error {
	userID := claims.UserID
	deviceID := ids[0]

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := sql.GetUserByID(tx, userID); err != nil {
			return err
		}

		var device models.MFADevice
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			Find(&device)
		if result.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
		}

		updates := make(map[string]any)

		if body.Name != nil {
			var existing models.MFADevice
			result = tx.Where("user_id = ? AND name = ? AND id != ? AND is_verified = ?",
				userID, *body.Name, deviceID, true).First(&existing)
			if result.RowsAffected > 0 {
				return apierrors.New(http.StatusConflict, apierrors.CodeMFADeviceNameExists)
			}
			updates["name"] = *body.Name
		}

		if body.IsDefault != nil && *body.IsDefault {
			if !device.IsVerified {
				return apierrors.New(http.StatusBadRequest, apierrors.CodeUnverifiedDeviceCannotDefault)
			}
			tx.Model(&models.MFADevice{}).
				Where("user_id = ? AND id != ?", userID, deviceID).
				Update("is_default", false)
			updates["is_default"] = true
		}

		if len(updates) > 0 {
			if err := tx.Model(&device).Updates(updates).Error; err != nil {
				return err
			}
		}

		action := models.Activity{
			Message: activity.MFADeviceUpdated,
			Object:  device.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     activity.MFADeviceUpdated,
				UserID:     userID.String(),
				ObjectType: rbac.ResourceMFADevice.String(),
				DeviceID:   deviceID.String(),
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log MFA device update activity", zap.Error(logErr))
		}

		logger.Info("MFA device updated",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()))

		return nil
	})
}

func (s MFAService) RemoveDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceRemoveBody,
) error {
	userID := claims.UserID
	deviceID := ids[0]

	user, err := sql.GetUserByID(s.DB, userID)
	if err != nil {
		return err
	}

	var device models.MFADevice
	result := s.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device)
	if result.RowsAffected == 0 {
		return apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
	}
	stepUpDone := device.IsVerified
	if stepUpDone {
		if err = s.verifyMFAStepUp(logger, &user, body.Password, body.Code); err != nil {
			return err
		}
	}

	var deviceName string

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var locked models.MFADevice
		lockResult := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			First(&locked)
		if lockResult.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeMFADeviceNotFound)
		}

		if locked.IsVerified && !stepUpDone {
			return apierrors.New(http.StatusBadRequest, apierrors.CodeBadRequest)
		}

		deviceName = locked.Name
		wasDefault := locked.IsDefault
		wasVerified := locked.IsVerified

		if delErr := tx.Delete(&locked).Error; delErr != nil {
			return delErr
		}

		if wasDefault && wasVerified {
			var nextDefaults []models.MFADevice
			tx.Where("user_id = ? AND is_verified = ?", userID, true).
				Order("created_at ASC").
				Limit(1).
				Find(&nextDefaults)
			if len(nextDefaults) > 0 {
				tx.Model(&nextDefaults[0]).Update("is_default", true)
			}
		}

		action := models.Activity{
			Message: activity.MFADeviceRemoved,
			Object:  locked.ToActivity(),
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:     activity.MFADeviceRemoved,
				UserID:     userID.String(),
				ObjectType: rbac.ResourceMFADevice.String(),
				DeviceID:   deviceID.String(),
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log MFA device removal activity", zap.Error(logErr))
		}

		logger.Info("MFA device removed",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()))

		return nil
	})

	if err != nil {
		return err
	}

	go func() {
		if notifyErr := s.Notifier.NotifyFromTemplate(
			user.Email,
			"MFA Device Removed - Safebucket",
			"mfa_device_removed",
			map[string]string{
				"DeviceName": deviceName,
				"WebURL":     s.AuthConfig.WebURL,
			},
		); notifyErr != nil {
			logger.Warn("Failed to send MFA device removal notification",
				zap.Error(notifyErr),
				zap.String("user_id", userID.String()),
				zap.String("email", user.Email))
		}
	}()

	return nil
}

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
	result := s.DB.Where("user_id = ? AND is_verified = ?", user.ID, true).Find(&devices)
	if result.Error != nil {
		logger.Error("Failed to load verified MFA devices for step-up", zap.Error(result.Error))
		return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}
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
		verifiedCount, err := sql.CountVerifiedMFADevices(s.DB, user.ID)
		if err != nil {
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
