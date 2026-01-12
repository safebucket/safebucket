package services

import (
	"strings"
	"time"

	"api/internal/activity"
	"api/internal/configuration"
	apierrors "api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AuthPasswordResetService struct {
	DB             *gorm.DB
	AuthConfig     models.AuthConfig
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
}

func NewAuthPasswordResetService(s AuthService) AuthPasswordResetService {
	return AuthPasswordResetService{
		DB:             s.DB,
		AuthConfig:     s.AuthConfig,
		Publisher:      s.Publisher,
		ActivityLogger: s.ActivityLogger,
	}
}

func (s AuthPasswordResetService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Validate[models.PasswordResetRequestBody]).
		Post("/", handlers.CreateHandler(s.RequestPasswordReset))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Validate[models.PasswordResetValidateBody]).
			Post("/validate", handlers.CreateHandler(s.ValidatePasswordReset))
		r.With(m.Validate[models.PasswordResetCompleteBody]).
			Post("/complete", handlers.CreateHandler(s.CompletePasswordReset))
	})

	return r
}

func (s AuthPasswordResetService) RequestPasswordReset(
	logger *zap.Logger,
	_ models.UserClaims,
	_ uuid.UUIDs,
	body models.PasswordResetRequestBody,
) (any, error) {
	var user models.User
	result := s.DB.Where("email = ? AND provider_type = ?", body.Email, models.LocalProviderType).
		First(&user)

	if result.RowsAffected == 0 {
		return nil, nil
	}

	secret, err := h.GenerateSecret()
	if err != nil {
		logger.Error("Failed to generate secret", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		logger.Error("Failed to hash secret", zap.Error(err))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	s.DB.Where("user_id = ? AND type = ?", user.ID, models.ChallengeTypePasswordReset).
		Delete(&models.Challenge{})

	expiresAt := time.Now().Add(configuration.SecurityChallengeExpirationMinutes * time.Minute)
	challenge := models.Challenge{
		Type:         models.ChallengeTypePasswordReset,
		UserID:       &user.ID,
		HashedSecret: hashedSecret,
		ExpiresAt:    &expiresAt,
		AttemptsLeft: configuration.SecurityChallengeMaxFailedAttempts,
	}

	result = s.DB.Create(&challenge)
	if result.Error != nil {
		logger.Error("Failed to create challenge", zap.Error(result.Error))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	event := events.NewPasswordResetChallenge(
		s.Publisher,
		secret,
		user.Email,
		challenge.ID.String(),
		s.AuthConfig.WebURL,
	)
	event.Trigger()

	return nil, nil
}

// ValidatePasswordReset verifies the reset code and returns either an MFA token
// (if user has MFA enabled) or a completion token (if no MFA).
func (s AuthPasswordResetService) ValidatePasswordReset(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.PasswordResetValidateBody,
) (models.AuthLoginResponse, error) {
	challengeID := ids[0]

	var challenge models.Challenge

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("User").
			Where("id = ? AND type = ?", challengeID, models.ChallengeTypePasswordReset).
			First(&challenge)

		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(400, "INVALID_REQUEST")
		}

		if challenge.User == nil {
			logger.Error("Challenge has no associated user")
			return apierrors.NewAPIError(400, "INVALID_REQUEST")
		}

		if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
			tx.Delete(&challenge)
			return apierrors.NewAPIError(400, "INVALID_REQUEST")
		}

		match, err := argon2id.ComparePasswordAndHash(
			strings.ToUpper(body.Code),
			challenge.HashedSecret,
		)
		if err != nil || !match {
			challenge.AttemptsLeft--

			if challenge.AttemptsLeft <= 0 {
				logger.Warn("Password reset challenge soft deleted due to too many failed attempts",
					zap.String("challenge_id", challenge.ID.String()),
					zap.String("user_id", challenge.UserID.String()),
					zap.Int("attempts_left", challenge.AttemptsLeft))
				tx.Delete(&challenge)

				return apierrors.NewAPIError(403, "CHALLENGE_LOCKED")
			}

			if updateErr := tx.Save(&challenge).Error; updateErr != nil {
				logger.Error("Failed to update attempts counter", zap.Error(updateErr))
				return updateErr
			}
			return apierrors.NewAPIError(401, "WRONG_CODE")
		}

		return nil
	})
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	user := challenge.User

	var userWithMFA models.User
	if err := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ?", user.ID).First(&userWithMFA).Error; err != nil {
		logger.Error("Failed to load user with MFA devices", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	if userWithMFA.HasMFAEnabled() {
		if err := s.DB.Model(&challenge).Update("status", models.ChallengeStatusValidated).Error; err != nil {
			logger.Error("Failed to update challenge status", zap.Error(err))
			return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		mfaToken, err := h.NewPasswordResetMFAToken(
			s.AuthConfig.JWTSecret,
			&userWithMFA,
		)
		if err != nil {
			logger.Error("Failed to generate MFA token", zap.Error(err))
			return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "MFA_TOKEN_GENERATION_FAILED")
		}

		devices := make([]models.MFADevice, 0)
		for _, device := range userWithMFA.GetVerifiedDevices() {
			devices = append(devices, models.MFADevice{
				ID:        device.ID,
				Name:      device.Name,
				IsDefault: device.IsDefault,
			})
		}

		action := models.Activity{
			Message: activity.PasswordResetCodeVerified,
			Object:  user.ToActivity(),
			Filter: activity.NewLogFilter(map[string]string{
				"action":       activity.PasswordResetCodeVerified,
				"user_id":      user.ID.String(),
				"challenge_id": challengeID.String(),
				"object_type":  "user",
				"mfa_required": "true",
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log password reset code verification", zap.Error(logErr))
		}

		logger.Info("Password reset code verified, MFA required",
			zap.String("user_id", user.ID.String()),
			zap.String("challenge_id", challengeID.String()))

		return models.AuthLoginResponse{
			MFARequired: true,
			MFAToken:    mfaToken,
			Devices:     devices,
		}, nil
	}

	completionToken, err := h.NewPasswordResetCompletionToken(
		s.AuthConfig.JWTSecret,
		&userWithMFA,
	)
	if err != nil {
		logger.Error("Failed to generate completion token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "COMPLETION_TOKEN_GENERATION_FAILED")
	}

	action := models.Activity{
		Message: activity.PasswordResetCodeVerified,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":       activity.PasswordResetCodeVerified,
			"user_id":      user.ID.String(),
			"challenge_id": challengeID.String(),
			"object_type":  "user",
			"mfa_required": "false",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log password reset code verification", zap.Error(logErr))
	}

	logger.Info("Password reset code verified, no MFA required",
		zap.String("user_id", user.ID.String()),
		zap.String("challenge_id", challengeID.String()))

	return models.AuthLoginResponse{
		MFARequired:     false,
		CompletionToken: completionToken,
	}, nil
}

// CompletePasswordReset applies the new password after verifying the completion token.
func (s AuthPasswordResetService) CompletePasswordReset(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.PasswordResetCompleteBody,
) (models.AuthLoginResponse, error) {
	challengeID := ids[0]

	claims, err := h.ParsePasswordResetCompletionToken(s.AuthConfig.JWTSecret, body.CompletionToken)
	if err != nil {
		logger.Warn("Invalid completion token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(403, "INVALID_COMPLETION_TOKEN")
	}

	var challenge models.Challenge
	result := s.DB.Preload("User").
		Where("id = ? AND type = ?", challengeID, models.ChallengeTypePasswordReset).
		First(&challenge)

	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	if challenge.UserID == nil || *challenge.UserID != claims.UserID {
		logger.Warn("User mismatch in password reset completion",
			zap.String("token_user_id", claims.UserID.String()),
			zap.String("challenge_user_id", challenge.UserID.String()))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	user := challenge.User

	var userWithMFA models.User
	if err := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ?", user.ID).First(&userWithMFA).Error; err != nil {
		logger.Error("Failed to load user with MFA devices", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}
	user = &userWithMFA

	hashedPassword, err := h.CreateHash(body.NewPassword)
	if err != nil {
		logger.Error("Failed to hash new password", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "PASSWORD_HASH_FAILED")
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(user).Update("hashed_password", hashedPassword).Error; err != nil {
			logger.Error("Failed to update password", zap.Error(err))
			return apierrors.NewAPIError(500, "PASSWORD_UPDATE_FAILED")
		}
		if err := tx.Delete(&challenge).Error; err != nil {
			logger.Error("Failed to delete challenge", zap.Error(err))
			return apierrors.NewAPIError(500, "CHALLENGE_CLEANUP_FAILED")
		}
		return nil
	})
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	resetDate := time.Now().Format("January 2, 2006 at 3:04 PM MST")
	successEvent := events.NewPasswordResetSuccess(
		s.Publisher,
		user.Email,
		s.AuthConfig.WebURL,
		resetDate,
	)
	successEvent.Trigger()

	action := models.Activity{
		Message: activity.PasswordResetCompleted,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":       activity.PasswordResetCompleted,
			"user_id":      user.ID.String(),
			"challenge_id": challengeID.String(),
			"object_type":  "user",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log password reset completion", zap.Error(logErr))
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "GENERATE_ACCESS_TOKEN_FAILED")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "GENERATE_REFRESH_TOKEN_FAILED")
	}

	logger.Info("Password reset completed successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("challenge_id", challengeID.String()))

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// HandleMFAVerification handles MFA verification for password reset flow.
func (s AuthPasswordResetService) HandleMFAVerification(
	logger *zap.Logger,
	user *models.User,
	deviceID string,
) (models.AuthLoginResponse, error) {
	var challenge models.Challenge
	result := s.DB.Where("user_id = ? AND type = ? AND status = ?",
		user.ID, models.ChallengeTypePasswordReset, models.ChallengeStatusValidated).
		First(&challenge)
	if result.RowsAffected == 0 {
		logger.Warn("Validated password reset challenge not found",
			zap.String("user_id", user.ID.String()))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	challengeID := challenge.ID

	completionToken, err := h.NewPasswordResetCompletionToken(
		s.AuthConfig.JWTSecret,
		user,
	)
	if err != nil {
		logger.Error("Failed to generate completion token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "COMPLETION_TOKEN_GENERATION_FAILED")
	}

	action := models.Activity{
		Message: activity.PasswordResetMFAVerified,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":       activity.PasswordResetMFAVerified,
			"user_id":      user.ID.String(),
			"challenge_id": challengeID.String(),
			"device_id":    deviceID,
			"object_type":  "user",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log password reset MFA verification", zap.Error(logErr))
	}

	logger.Info("Password reset MFA verification successful",
		zap.String("user_id", user.ID.String()),
		zap.String("device_id", deviceID),
		zap.String("challenge_id", challengeID.String()))

	return models.AuthLoginResponse{
		CompletionToken: completionToken,
		PasswordReset:   true,
	}, nil
}
