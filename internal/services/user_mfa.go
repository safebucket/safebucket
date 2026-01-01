package services

import (
	"strings"
	"time"

	"api/internal/cache"
	"api/internal/configuration"
	apierrors "api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/notifier"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserMFAService struct {
	DB                 *gorm.DB
	Cache              cache.ICache
	MFAEncryptionKey   string
	JWTSecret          string
	AccessTokenExpiry  int
	RefreshTokenExpiry int
	Publisher          messaging.IPublisher
	WebURL             string
	Notifier           notifier.INotifier
}

func (s UserMFAService) Routes() chi.Router {
	r := chi.NewRouter()

	// Device management routes
	r.Route("/devices", func(r chi.Router) {
		r.With(m.AuthorizeSelfOrAdmin(0)).
			Get("/", handlers.GetOneHandler(s.ListDevices))

		r.With(m.AuthorizeSelfOrAdmin(0)).
			With(m.Validate[models.MFADeviceSetupBody]).
			Post("/", handlers.CreateHandler(s.AddDevice))

		r.Route("/{id1}", func(r chi.Router) {
			r.With(m.AuthorizeSelfOrAdmin(0)).
				Get("/", handlers.GetOneHandler(s.GetDevice))

			r.With(m.AuthorizeSelfOrAdmin(0)).
				With(m.Validate[models.MFADeviceUpdateBody]).
				Patch("/", handlers.BodyHandler(s.UpdateDevice))

			r.With(m.AuthorizeSelfOrAdmin(0)).
				With(m.Validate[models.MFADeviceRemoveBody]).
				Delete("/", handlers.BodyHandler(s.RemoveDevice))

			r.With(m.AuthorizeSelfOrAdmin(0)).
				With(m.Validate[models.MFADeviceVerifyBody]).
				Post("/verify", handlers.CreateHandler(s.VerifyDevice))
		})
	})

	// Reset flow
	r.Route("/reset", func(r chi.Router) {
		r.With(m.AuthorizeSelfOrAdmin(0)).
			With(m.Validate[models.MFAResetRequestBody]).
			Post("/", handlers.CreateHandler(s.RequestMFAReset))

		r.With(m.AuthorizeSelfOrAdmin(0)).
			With(m.Validate[models.MFAResetVerifyBody]).
			Post("/{id1}", handlers.CreateHandler(s.VerifyMFAReset))
	})

	return r
}

// ListDevices returns all MFA devices for a user.
func (s UserMFAService) ListDevices(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (models.MFADevicesListResponse, error) {
	userID := ids[0]

	var devices []models.MFADevice
	result := s.DB.Where("user_id = ?", userID).
		Order("is_default DESC, created_at ASC").
		Find(&devices)
	if result.Error != nil {
		return models.MFADevicesListResponse{}, result.Error
	}

	responses := make([]models.MFADeviceResponse, len(devices))
	verifiedCount := 0
	for i, d := range devices {
		responses[i] = d.ToResponse()
		if d.IsVerified {
			verifiedCount++
		}
	}

	return models.MFADevicesListResponse{
		Devices:     responses,
		MFAEnabled:  verifiedCount > 0,
		DeviceCount: len(devices),
		MaxDevices:  models.MaxMFADevicesPerUser,
	}, nil
}

// AddDevice initiates MFA device setup.
func (s UserMFAService) AddDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceSetupBody,
) (models.MFADeviceSetupResponse, error) {
	userID := ids[0]

	// Get user (must be local provider)
	var user models.User
	result := s.DB.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).First(&user)
	if result.RowsAffected == 0 {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	// Default device name if not provided
	deviceName := body.Name
	if deviceName == "" {
		deviceName = "Authenticator"
	}

	// Check device limit
	var count int64
	s.DB.Model(&models.MFADevice{}).Where("user_id = ?", userID).Count(&count)
	if count >= int64(models.MaxMFADevicesPerUser) {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(400, "MAX_MFA_DEVICES_REACHED")
	}

	// Check for duplicate name
	var existing models.MFADevice
	result = s.DB.Where("user_id = ? AND name = ?", userID, deviceName).Find(&existing)
	if result.RowsAffected > 0 {
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(409, "MFA_DEVICE_NAME_EXISTS")
	}

	// Generate TOTP secret
	totpKey, err := h.GenerateTOTPSecret(user.Email)
	if err != nil {
		logger.Error("Failed to generate TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
	}

	// Encrypt secret
	encryptedSecret, err := h.EncryptSecret(totpKey.Secret, []byte(s.MFAEncryptionKey))
	if err != nil {
		logger.Error("Failed to encrypt TOTP secret", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
	}

	// Create unverified device (is_default will be set when verified)
	device := models.MFADevice{
		UserID:          userID,
		Name:            deviceName,
		Type:            models.MFADeviceTypeTOTP,
		SecretEncrypted: encryptedSecret,
		IsDefault:       false, // Will be set to true when verified if no other default exists
		IsVerified:      false,
	}

	if err = s.DB.Create(&device).Error; err != nil {
		logger.Error("Failed to create MFA device", zap.Error(err))
		return models.MFADeviceSetupResponse{}, apierrors.NewAPIError(500, "MFA_SETUP_FAILED")
	}

	logger.Info("MFA device setup initiated",
		zap.String("user_id", userID.String()),
		zap.String("device_id", device.ID.String()),
		zap.String("device_name", deviceName),
		zap.String("initiated_by", claims.UserID.String()))

	return models.MFADeviceSetupResponse{
		DeviceID:  device.ID,
		Secret:    totpKey.Secret,
		QRCodeURI: totpKey.URL,
		Issuer:    configuration.AppName,
	}, nil
}

// GetDevice returns a specific MFA device.
func (s UserMFAService) GetDevice(
	_ *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) (models.MFADeviceResponse, error) {
	userID := ids[0]
	deviceID := ids[1]

	var device models.MFADevice
	result := s.DB.Where("id = ? AND user_id = ?", deviceID, userID).First(&device)
	if result.RowsAffected == 0 {
		return models.MFADeviceResponse{}, apierrors.NewAPIError(404, "MFA_DEVICE_NOT_FOUND")
	}

	return device.ToResponse(), nil
}

// VerifyDevice verifies a TOTP code and enables the device.
func (s UserMFAService) VerifyDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceVerifyBody,
) (interface{}, error) {
	userID := ids[0]
	deviceID := ids[1]

	var accessToken, refreshToken string
	var user models.User
	var deviceName string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).Find(&user)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "USER_NOT_FOUND")
		}

		// Lock device row
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

		// Store device name for notification
		deviceName = device.Name

		// Decrypt and validate TOTP
		secret, err := h.DecryptSecret(device.SecretEncrypted, []byte(s.MFAEncryptionKey))
		if err != nil {
			logger.Error("Failed to decrypt TOTP secret", zap.Error(err))
			return apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
		}

		if !h.ValidateTOTPCode(secret, body.Code) {
			logger.Warn("MFA device verification failed - invalid code",
				zap.String("user_id", userID.String()),
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(401, "INVALID_MFA_CODE")
		}

		// Check replay protection (per device)
		used, err := s.Cache.IsTOTPCodeUsed(deviceID.String(), body.Code)
		if err != nil {
			logger.Error("Failed to check TOTP code usage", zap.Error(err))
			return apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
		}
		if used {
			logger.Warn("TOTP code replay attempt detected",
				zap.String("device_id", deviceID.String()))
			return apierrors.NewAPIError(401, "INVALID_MFA_CODE")
		}

		if err = s.Cache.MarkTOTPCodeUsed(deviceID.String(), body.Code); err != nil {
			logger.Error("Failed to mark TOTP code as used", zap.Error(err))
			return apierrors.NewAPIError(500, "MFA_VERIFICATION_FAILED")
		}

		// Check if there's already a verified default device
		var existingDefaultCount int64
		tx.Model(&models.MFADevice{}).
			Where("user_id = ? AND is_verified = ? AND is_default = ? AND id != ?",
				userID, true, true, deviceID).
			Count(&existingDefaultCount)

		// This device should be default only if no other verified default exists
		shouldBeDefault := existingDefaultCount == 0

		// Enable device
		now := time.Now()
		if err = tx.Model(&device).Updates(map[string]interface{}{
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

	// Reload user with MFA devices for token generation
	s.DB.Preload("MFADevices", "is_verified = ?", true).First(&user, userID)
	provider := string(user.ProviderType)
	accessToken, err = h.NewAccessToken(s.JWTSecret, &user, provider, s.AccessTokenExpiry)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "TOKEN_GENERATION_FAILED")
	}

	refreshToken, err = h.NewRefreshToken(s.JWTSecret, &user, provider, s.RefreshTokenExpiry)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "TOKEN_GENERATION_FAILED")
	}

	logger.Info("MFA device verified and enabled",
		zap.String("user_id", userID.String()),
		zap.String("device_id", deviceID.String()),
		zap.String("verified_by", claims.UserID.String()))

	// Send notification email (outside transaction)
	go func() {
		if notifyErr := s.Notifier.NotifyFromTemplate(
			user.Email,
			"New MFA Device Enrolled - Safebucket",
			"mfa_device_enrolled",
			map[string]string{
				"DeviceName": deviceName,
				"WebURL":     s.WebURL,
			},
		); notifyErr != nil {
			logger.Warn("Failed to send MFA device enrollment notification",
				zap.Error(notifyErr),
				zap.String("user_id", userID.String()),
				zap.String("email", user.Email))
		}
	}()

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// UpdateDevice updates device properties (name, primary status).
func (s UserMFAService) UpdateDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceUpdateBody,
) error {
	userID := ids[0]
	deviceID := ids[1]

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

		updates := make(map[string]interface{})

		if body.Name != nil {
			// Check for duplicate name
			var existing models.MFADevice
			result = tx.Where("user_id = ? AND name = ? AND id != ?",
				userID, *body.Name, deviceID).First(&existing)
			if result.RowsAffected > 0 {
				return apierrors.NewAPIError(409, "MFA_DEVICE_NAME_EXISTS")
			}
			updates["name"] = *body.Name
		}

		if body.IsDefault != nil && *body.IsDefault {
			if !device.IsVerified {
				return apierrors.NewAPIError(400, "UNVERIFIED_DEVICE_CANNOT_BE_DEFAULT")
			}
			// Clear other defaults
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

		logger.Info("MFA device updated",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()),
			zap.String("updated_by", claims.UserID.String()))

		return nil
	})
}

// RemoveDevice removes an MFA device after verifying user password.
func (s UserMFAService) RemoveDevice(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFADeviceRemoveBody,
) error {
	userID := ids[0]
	deviceID := ids[1]

	var user models.User
	var deviceName string

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Fetch user first for password verification
		result := tx.Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).First(&user)
		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "USER_NOT_FOUND")
		}

		// Verify password before allowing device removal
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

		// Store device name for notification
		deviceName = device.Name

		wasDefault := device.IsDefault
		wasVerified := device.IsVerified

		// Delete device
		if err = tx.Delete(&device).Error; err != nil {
			return err
		}

		// If this was default, promote another verified device
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

		logger.Info("MFA device removed",
			zap.String("user_id", userID.String()),
			zap.String("device_id", deviceID.String()),
			zap.String("removed_by", claims.UserID.String()))

		return nil
	})

	if err != nil {
		return err
	}

	// Send notification email (outside transaction)
	go func() {
		if notifyErr := s.Notifier.NotifyFromTemplate(
			user.Email,
			"MFA Device Removed - Safebucket",
			"mfa_device_removed",
			map[string]string{
				"DeviceName": deviceName,
				"WebURL":     s.WebURL,
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

// RequestMFAReset initiates MFA reset by sending an email challenge.
func (s UserMFAService) RequestMFAReset(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.MFAResetRequestBody,
) (models.MFAResetRequestResponse, error) {
	userID := ids[0]

	var user models.User
	result := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ? AND provider_type = ?", userID, models.LocalProviderType).First(&user)
	if result.RowsAffected == 0 {
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(404, "USER_NOT_FOUND")
	}

	// Check if user has any verified devices
	if !user.HasMFAEnabled() {
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(409, "MFA_NOT_ENABLED")
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.HashedPassword)
	if err != nil || !match {
		logger.Warn("MFA reset request failed - invalid password",
			zap.String("user_id", userID.String()))
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(401, "INVALID_PASSWORD")
	}

	secret, err := h.GenerateSecret()
	if err != nil {
		logger.Error("Failed to generate challenge secret", zap.Error(err))
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(500, "MFA_RESET_REQUEST_FAILED")
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		logger.Error("Failed to hash challenge secret", zap.Error(err))
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(500, "MFA_RESET_REQUEST_FAILED")
	}

	s.DB.Where("user_id = ? AND type = ?", user.ID, models.ChallengeTypeMFAReset).
		Delete(&models.Challenge{})

	expiresAt := time.Now().Add(configuration.SecurityChallengeExpirationMinutes * time.Minute)
	challenge := models.Challenge{
		Type:         models.ChallengeTypeMFAReset,
		UserID:       &user.ID,
		HashedSecret: hashedSecret,
		ExpiresAt:    &expiresAt,
		AttemptsLeft: configuration.SecurityChallengeMaxFailedAttempts,
	}

	result = s.DB.Create(&challenge)
	if result.Error != nil {
		logger.Error("Failed to create MFA reset challenge", zap.Error(result.Error))
		return models.MFAResetRequestResponse{}, apierrors.NewAPIError(500, "MFA_RESET_REQUEST_FAILED")
	}

	event := events.NewMFAResetChallenge(
		s.Publisher,
		secret,
		user.Email,
		challenge.ID.String(),
		s.WebURL,
	)
	event.Trigger()

	logger.Info("MFA reset requested",
		zap.String("user_id", userID.String()),
		zap.String("challenge_id", challenge.ID.String()))

	return models.MFAResetRequestResponse{
		ChallengeID: challenge.ID.String(),
	}, nil
}

// VerifyMFAReset verifies the reset challenge and disables all MFA.
func (s UserMFAService) VerifyMFAReset(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.MFAResetVerifyBody,
) (interface{}, error) {
	userID := ids[0]
	challengeID := ids[1]

	var challenge models.Challenge
	result := s.DB.Preload("User").
		Where("id = ? AND type = ? AND user_id = ?", challengeID, models.ChallengeTypeMFAReset, userID).
		First(&challenge)

	if result.RowsAffected == 0 {
		return nil, apierrors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return nil, apierrors.NewAPIError(410, "CHALLENGE_EXPIRED")
	}

	match, err := argon2id.ComparePasswordAndHash(
		strings.ToUpper(body.Code),
		challenge.HashedSecret,
	)
	if err != nil || !match {
		updateResult := s.DB.Model(&challenge).
			Where("attempts_left > 0").
			Update("attempts_left", gorm.Expr("attempts_left - 1"))

		if updateResult.Error != nil {
			logger.Error("Failed to update attempts counter", zap.Error(updateResult.Error))
		}

		var updatedChallenge models.Challenge
		s.DB.First(&updatedChallenge, challenge.ID)

		if updatedChallenge.AttemptsLeft <= 0 {
			logger.Warn("MFA reset challenge deleted due to too many failed attempts",
				zap.String("challenge_id", challenge.ID.String()),
				zap.String("user_id", userID.String()))
			s.DB.Delete(&challenge)
			return nil, apierrors.NewAPIError(403, "CHALLENGE_LOCKED")
		}

		return nil, apierrors.NewAPIError(401, "INVALID_RESET_CODE")
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		// Delete all MFA devices
		if txErr := tx.Where("user_id = ?", userID).Delete(&models.MFADevice{}).Error; txErr != nil {
			return txErr
		}

		deleteResult := tx.Delete(&challenge)
		if deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			return apierrors.NewAPIError(500, "MFA_RESET_FAILED")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	logger.Info("MFA reset completed - all devices removed",
		zap.String("user_id", userID.String()),
		zap.String("reset_by", claims.UserID.String()))

	return nil, nil
}
