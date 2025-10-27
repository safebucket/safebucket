package services

import (
	"strings"
	"time"

	"api/internal/activity"
	"api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/sql"
	"api/internal/storage"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InviteService struct {
	DB             *gorm.DB
	Storage        storage.IStorage
	JWTSecret      string
	Publisher      messaging.IPublisher
	Providers      configuration.Providers
	ActivityLogger activity.IActivityLogger
	WebURL         string
}

func (s InviteService) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Validate[models.InviteChallengeCreateBody]).
			Post("/challenges", handlers.CreateHandler(s.CreateInviteChallenge))

		r.Route("/challenges/{id1}", func(r chi.Router) {
			r.With(m.Validate[models.InviteChallengeValidateBody]).
				Post("/validate", handlers.CreateHandler(s.ValidateInviteChallenge))
		})
	})

	return r
}

func (s InviteService) CreateInviteChallenge(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.InviteChallengeCreateBody,
) (interface{}, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return nil, errors.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return nil, errors.NewAPIError(403, "FORBIDDEN")
	}

	inviteID := ids[0]
	var invite models.Invite
	result := s.DB.Preload("User").Where("id = ?", inviteID).First(&invite)

	if result.RowsAffected == 0 {
		return nil, errors.NewAPIError(404, "INVITE_NOT_FOUND")
	}

	if invite.Email != body.Email {
		logger.Warn("Invite email mismatch attempt detected",
			zap.String("invite_id", inviteID.String()),
			zap.String("provided_email", body.Email))
		return nil, errors.NewAPIError(404, "INVITE_NOT_FOUND")
	}

	s.DB.Where("invite_id = ? AND type = ?", invite.ID, models.ChallengeTypeInvite).
		Delete(&models.Challenge{})

	secret, err := h.GenerateSecret()
	if err != nil {
		return nil, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		return nil, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
	}

	// Create challenge with expiration and attempt limiting
	expiresAt := time.Now().Add(configuration.SecurityChallengeExpirationMinutes * time.Minute)
	challenge := models.Challenge{
		Type:         models.ChallengeTypeInvite,
		InviteID:     &invite.ID,
		HashedSecret: hashedSecret,
		ExpiresAt:    &expiresAt,
		AttemptsLeft: configuration.SecurityChallengeMaxFailedAttempts,
	}

	result = s.DB.Create(&challenge)
	if result.Error != nil {
		return nil, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
	}

	event := events.NewChallengeUserInvite(
		s.Publisher,
		secret,
		invite.Email,
		invite.User.Email,
		inviteID.String(),
		challenge.ID.String(),
		s.WebURL,
	)
	event.Trigger()

	// Don't return challenge ID - it's only available in the email notification
	return nil, nil
}

func (s InviteService) ValidateInviteChallenge(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.InviteChallengeValidateBody,
) (models.AuthLoginResponse, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	inviteID := ids[0]
	challengeID := ids[1]

	var challenge models.Challenge

	result := s.DB.Preload("Invite").
		Where("id = ? AND invite_id = ? AND type = ?", challengeID, inviteID, models.ChallengeTypeInvite).
		First(&challenge)

	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, errors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
	}

	if challenge.Invite == nil {
		logger.Error("Challenge has no associated invite")
		return models.AuthLoginResponse{}, errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
		s.DB.Delete(&challenge)
		return models.AuthLoginResponse{}, errors.NewAPIError(410, "CHALLENGE_EXPIRED")
	}

	if !h.IsDomainAllowed(
		challenge.Invite.Email,
		s.Providers[string(models.LocalProviderType)].Domains,
	) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	match, err := argon2id.ComparePasswordAndHash(
		strings.ToUpper(body.Code),
		challenge.HashedSecret,
	)
	if err != nil || !match {
		challenge.AttemptsLeft--

		// Soft delete if max attempts reached
		if challenge.AttemptsLeft <= 0 {
			logger.Warn("Invite challenge soft deleted due to too many failed attempts",
				zap.String("challenge_id", challenge.ID.String()),
				zap.String("invite_id", challenge.InviteID.String()),
				zap.Int("attempts_left", challenge.AttemptsLeft))
			s.DB.Delete(&challenge)
			return models.AuthLoginResponse{}, errors.NewAPIError(403, "CHALLENGE_LOCKED")
		}

		// Update attempts counter
		if updateErr := s.DB.Model(&challenge).Update("attempts_left", challenge.AttemptsLeft).Error; updateErr != nil {
			logger.Error("Failed to update attempts counter", zap.Error(updateErr))
		}

		return models.AuthLoginResponse{}, errors.NewAPIError(401, "WRONG_CODE")
	}

	// Check if user already exists
	newUser := models.User{
		Email:        challenge.Invite.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
	}

	result = s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected > 0 {
		return models.AuthLoginResponse{}, errors.NewAPIError(401, "USER_ALREADY_EXISTS")
	}

	hashedPassword, err := h.CreateHash(body.NewPassword)
	if err != nil {
		logger.Error("Failed to hash password", zap.Error(err))
		return models.AuthLoginResponse{}, errors.NewAPIError(500, "PASSWORD_HASH_FAILED")
	}

	newUser.HashedPassword = hashedPassword
	newUser.Role = models.RoleGuest

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err = sql.CreateUserWithInvites(logger, tx, &newUser); err != nil {
			return errors.NewAPIError(500, "USER_CREATION_FAILED")
		}

		if deleteResult := tx.Delete(&challenge); deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			return errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		return models.AuthLoginResponse{}, errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	welcomeEvent := events.NewUserWelcome(
		s.Publisher,
		newUser.Email,
		s.WebURL,
	)
	welcomeEvent.Trigger()

	accessToken, err := h.NewAccessToken(s.JWTSecret, &newUser, string(models.LocalProviderType))
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.AuthLoginResponse{}, errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(s.JWTSecret, &newUser, string(models.LocalProviderType))
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
