package services

import (
	"time"

	"github.com/safebucket/safebucket/internal/activity"
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
				Post("/verify", handlers.CreateHandler(s.VerifyDevice))
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

	var user models.User
	result := s.DB.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).First(&user)
	if result.RowsAffected == 0 {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	var count int64
	result = s.DB.Model(&models.MFADevice{}).Where("user_id = ?", userID).Count(&count)
	if result.Error != nil {
		logger.Error("Failed to count MFA devices", zap.Error(result.Error))
		return models.MFADeviceSetupResponse{}, result.Error
	}
	if count >= int64(configuration.MaxMFADevicesPerUser) {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(400, "MAX_MFA_DEVICES_REACHED")
	}

	isRestricted := claims.AudienceString() == configuration.AudienceMFALogin ||
		claims.AudienceString() == configuration.AudienceMFAReset

	if isRestricted {
		var verifiedCount int64
		result = s.DB.Model(&models.MFADevice{}).
			Where("user_id = ? AND is_verified = ?", userID, true).
			Count(&verifiedCount)
		if result.Error != nil {
			logger.Error("Failed to count verified MFA devices", zap.Error(result.Error))
			return models.MFADeviceSetupResponse{}, result.Error
		}
		if verifiedCount > 0 {
			logger.Warn("restricted token used for non-initial device setup",
				zap.String("userID", claims.UserID.String()),
				zap.Int64("verifiedDeviceCount", verifiedCount))
			return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(403, "MFA_SETUP_RESTRICTED")
		}
	} else {
		if body.Password == "" {
			return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(400, "BAD_REQUEST")
		}

		match, err := argon2id.ComparePasswordAndHash(body.Password, user.HashedPassword)
		if err != nil {
			logger.Error("failed to compare password and hash", zap.Error(err))
			return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(400, "BAD_REQUEST")
		}
		if !match {
			logger.Warn("invalid password provided for device enrollment", zap.String("userID", claims.UserID.String()))
			return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(401, "INVALID_PASSWORD")
		}
	}

	var existing models.MFADevice
	result = s.DB.Where("user_id = ? AND name = ? AND is_verified = ?", userID, body.Name, true).Find(&existing)
	if result.RowsAffected > 0 {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(409, "MFA_DEVICE_NAME_EXISTS")
	}

	totpKey, err := h.GenerateTOTPSecret(user.Email)
	if err != nil {
		logger.Error("Failed to generate TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
	}

	encryptedSecret, err := h.EncryptSecret(totpKey.Secret, []byte(s.AuthConfig.MFAEncryptionKey))
	if err != nil {
		logger.Error("Failed to encrypt TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
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
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
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
		return models.MFADevice{}, apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
	}

	return device, nil
}

func (s MFAService) VerifyDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceVerifyBody,
) (any, error) {
	userID := claims.UserID
	deviceID := ids[0]

	var user models.User
	var deviceName string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).Find(&user)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "USER_NOT_FOUND")
		}

		var device models.MFADevice
		result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			First(&device)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
		}

		if device.IsVerified {
			return apierrors.NewAPIError(409, "MFA_DEVICE_ALREADY_VERIFIED")
		}

		deviceName = device.Name

		secret, err := h.DecryptSecret(device.EncryptedSecret, []byte(s.AuthConfig.MFAEncryptionKey))
		if err != nil {
			logger.Error("Failed to decrypt TOTP secret", zap.Error(err))
			return apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
		}

		attempts, err := cache.GetMFAAttempts(s.Cache, userID.String())
		if err != nil {
			logger.Error("Rate limit check failed - denying request", zap.Error(err))
			return apierrors.NewAPIError(503, "SERVICE_UNAVAILABLE")
		}
		if attempts >= configuration.MFAMaxAttempts {
			logger.Warn("MFA device verification rate limited",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(429, "MFA_RATE_LIMITED")
		}

		if !h.ValidateTOTPCode(secret, body.Code) {
			if incErr := cache.IncrementMFAAttempts(s.Cache, userID.String()); incErr != nil {
				logger.Error("Failed to increment MFA attempts", zap.Error(incErr))
			}

			logger.Warn("MFA device verification failed - invalid code",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(401, "INVALID_MFA_CODE")
		}

		if err = cache.ResetMFAAttempts(s.Cache, userID.String()); err != nil {
			logger.Error("Failed to reset MFA attempts", zap.Error(err))
		}

		unused, err := cache.MarkTOTPCodeUsed(s.Cache, deviceID.String(), body.Code)
		if err != nil {
			logger.Error("Failed to mark TOTP code as used", zap.Error(err))
			return apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
		}
		if !unused {
			logger.Warn("TOTP code replay attempt detected",
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(401, "INVALID_MFA_CODE")
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
		return nil, err
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
			s.AuthConfig.JWTSecret,
			&user,
			configuration.AudienceMFAReset,
			true,
			claims.ChallengeID,
		)
		if err != nil {
			logger.Error("Failed to generate restricted access token", zap.Error(err))
			return nil, apierrors.NewAPIError(500, "TOKEN_GENERATION_FAILED")
		}

		logger.Info("MFA device verified (password reset flow)",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()))

		return models.AuthLoginResponse{
			AccessToken: restrictedToken,
			MFARequired: false,
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
		return nil, err
	}

	if sessionErr := cache.CreateSession(s.Cache, userID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return nil, apierrors.ErrInternalServer
	}

	return tokens, nil
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
		var user models.User
		result := tx.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).Find(&user)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "USER_NOT_FOUND")
		}

		var device models.MFADevice
		result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			Find(&device)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
		}

		updates := make(map[string]any)

		if body.Name != nil {
			var existing models.MFADevice
			result = tx.Where("user_id = ? AND name = ? AND id != ? AND is_verified = ?",
				userID, *body.Name, deviceID, true).First(&existing)
			if result.RowsAffected > 0 {
				return apierrors.NewAPIError(409, "MFA_DEVICE_NAME_EXISTS")
			}
			updates["name"] = *body.Name
		}

		if body.IsDefault != nil && *body.IsDefault {
			if !device.IsVerified {
				return apierrors.NewAPIError(400, "UNVERIFIED_DEVICE_CANNOT_BE_DEFAULT")
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

	var user models.User
	var deviceName string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).First(&user)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "USER_NOT_FOUND")
		}

		match, err := argon2id.ComparePasswordAndHash(body.Password, user.HashedPassword)
		if err != nil {
			logger.Error("Failed to verify password", zap.Error(err))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
		if !match {
			logger.Warn("MFA device removal failed - invalid password",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(401, "INVALID_PASSWORD")
		}

		var device models.MFADevice
		result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", deviceID, userID).
			First(&device)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
		}

		deviceName = device.Name
		wasDefault := device.IsDefault
		wasVerified := device.IsVerified

		if err = tx.Delete(&device).Error; err != nil {
			return err
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
			Object:  device.ToActivity(),
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
