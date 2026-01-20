package services

import (
	"strconv"
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

// ValidatePasswordReset verifies the reset code and returns a restricted access token.
// If user has MFA enabled, frontend should verify MFA before completing password reset.
// Frontend fetches devices and determines MFA state by checking if devices list is empty.
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
	if err = s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ?", user.ID).First(&userWithMFA).Error; err != nil {
		logger.Error("Failed to load user with MFA devices", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	// Generate restricted access token for password reset flow
	// Include challenge ID in token to validate later (replaces status column check)
	restrictedToken, tokenErr := h.NewRestrictedAccessToken(
		s.AuthConfig.JWTSecret,
		&userWithMFA,
		configuration.AudienceMFAReset,
		false,
		&challengeID,
	)
	if tokenErr != nil {
		logger.Error("Failed to generate restricted access token", zap.Error(tokenErr))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	hasMFA := userWithMFA.HasMFAEnabled()

	action := models.Activity{
		Message: activity.PasswordResetCodeVerified,
		Object:  user.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":       activity.PasswordResetCodeVerified,
			"user_id":      user.ID.String(),
			"challenge_id": challengeID.String(),
			"object_type":  "user",
			"mfa_required": strconv.FormatBool(hasMFA),
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log password reset code verification", zap.Error(logErr))
	}

	logger.Info("Password reset code verified",
		zap.String("user_id", user.ID.String()),
		zap.String("challenge_id", challengeID.String()),
		zap.Bool("mfa_required", hasMFA))

	return models.AuthLoginResponse{
		AccessToken: restrictedToken,
		MFARequired: hasMFA,
	}, nil
}

// CompletePasswordReset applies the new password.
// Authorization is handled via restricted access token in Authorization header.
// For users with MFA, they must have verified MFA via /auth/mfa/verify first.
func (s AuthPasswordResetService) CompletePasswordReset(
	logger *zap.Logger,
	claims models.UserClaims,
	ids uuid.UUIDs,
	body models.PasswordResetCompleteBody,
) (models.AuthLoginResponse, error) {
	challengeID := ids[0]

	// Verify the JWT contains a challenge_id that matches the URL
	// This proves the code was validated (JWT only issued after successful validation)
	if claims.ChallengeID == nil || *claims.ChallengeID != challengeID {
		logger.Warn("Challenge ID mismatch in password reset completion",
			zap.String("url_challenge_id", challengeID.String()),
			zap.Any("jwt_challenge_id", claims.ChallengeID))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	challenge, err := s.getChallenge(logger, challengeID, claims.UserID)
	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	user := challenge.User

	var userWithMFA models.User
	if dbErr := s.DB.Preload("MFADevices", "is_verified = ?", true).
		Where("id = ?", user.ID).First(&userWithMFA).Error; dbErr != nil {
		logger.Error("Failed to load user with MFA devices", zap.Error(dbErr))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}
	user = &userWithMFA

	// CRITICAL: Prevent MFA bypass
	// If user has MFA enabled, token MUST indicate MFA is verified
	// Note: Audience validation (cross-flow attack prevention) is handled by middleware
	if userWithMFA.HasMFAEnabled() && !claims.MFA {
		logger.Warn("MFA bypass attempt in password reset",
			zap.String("user_id", user.ID.String()))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(403, "MFA_REQUIRED")
	}

	hashedPassword, err := h.CreateHash(body.NewPassword)
	if err != nil {
		logger.Error("Failed to hash new password", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if updateErr := tx.Model(user).Update("hashed_password", hashedPassword).Error; updateErr != nil {
			logger.Error("Failed to update password", zap.Error(updateErr))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}
		if deleteErr := tx.Delete(&challenge).Error; deleteErr != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteErr))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
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
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		user,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	logger.Info("Password reset completed successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("challenge_id", challengeID.String()))

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// getChallenge retrieves a challenge for the given user.
// Note: Status check is no longer needed - the JWT's challenge_id proves the code was validated.
func (s AuthPasswordResetService) getChallenge(
	_ *zap.Logger,
	challengeID uuid.UUID,
	userID uuid.UUID,
) (*models.Challenge, error) {
	var challenge models.Challenge
	result := s.DB.Preload("User").
		Where("id = ? AND type = ? AND user_id = ?",
			challengeID, models.ChallengeTypePasswordReset, userID).
		First(&challenge)

	if result.RowsAffected == 0 {
		return nil, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return nil, apierrors.NewAPIError(400, "INVALID_REQUEST")
	}

	return &challenge, nil
}
