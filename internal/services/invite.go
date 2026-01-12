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
	"api/internal/sql"
	"api/internal/storage"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InviteService struct {
	DB             *gorm.DB
	Storage        storage.IStorage
	AuthConfig     models.AuthConfig
	Publisher      messaging.IPublisher
	Providers      configuration.Providers
	ActivityLogger activity.IActivityLogger
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

func (s InviteService) handleInviteChallengeFailedAttempt(
	logger *zap.Logger,
	tx *gorm.DB,
	challenge *models.Challenge,
	inviteID uuid.UUID,
) error {
	challenge.AttemptsLeft--

	if challenge.AttemptsLeft <= 0 {
		logger.Warn("Invite challenge soft deleted due to too many failed attempts",
			zap.String("challenge_id", challenge.ID.String()),
			zap.String("invite_id", challenge.InviteID.String()),
			zap.Int("attempts_left", challenge.AttemptsLeft))
		tx.Delete(challenge)

		action := models.Activity{
			Message: activity.InviteChallengeLocked,
			Object:  nil,
			Filter: activity.NewLogFilter(map[string]string{
				"action":       activity.InviteChallengeLocked,
				"challenge_id": challenge.ID.String(),
				"invite_id":    inviteID.String(),
				"object_type":  "challenge",
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log invite challenge lockout", zap.Error(logErr))
		}

		return apierrors.NewAPIError(403, "CHALLENGE_LOCKED")
	}

	if updateErr := tx.Save(challenge).Error; updateErr != nil {
		logger.Error("Failed to update attempts counter", zap.Error(updateErr))
		return updateErr
	}

	action := models.Activity{
		Message: activity.InviteChallengeAttemptFailed,
		Object:  nil,
		Filter: activity.NewLogFilter(map[string]string{
			"action":        activity.InviteChallengeAttemptFailed,
			"challenge_id":  challenge.ID.String(),
			"invite_id":     inviteID.String(),
			"attempts_left": strconv.Itoa(challenge.AttemptsLeft),
			"object_type":   "challenge",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log failed invite attempt", zap.Error(logErr))
	}

	return apierrors.NewAPIError(401, "WRONG_CODE")
}

func (s InviteService) createUserFromInvite(
	logger *zap.Logger,
	invite *models.Invite,
	challenge *models.Challenge,
	password string,
	inviteID uuid.UUID,
) (models.AuthLoginResponse, error) {
	newUser := models.User{
		Email:        invite.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
	}

	result := s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected > 0 {
		return models.AuthLoginResponse{}, apierrors.NewAPIError(401, "USER_ALREADY_EXISTS")
	}

	hashedPassword, err := h.CreateHash(password)
	if err != nil {
		logger.Error("Failed to hash password", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "PASSWORD_HASH_FAILED")
	}

	newUser.HashedPassword = hashedPassword
	newUser.Role = models.RoleGuest

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err = sql.CreateUserWithInvites(logger, tx, &newUser); err != nil {
			logger.Error("Failed to create user with invites", zap.Error(err))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		if deleteResult := tx.Delete(challenge); deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	welcomeEvent := events.NewUserWelcome(
		s.Publisher,
		newUser.Email,
		s.AuthConfig.WebURL,
	)
	welcomeEvent.Trigger()

	action := models.Activity{
		Message: activity.InviteAccepted,
		Object:  newUser.ToActivity(),
		Filter: activity.NewLogFilter(map[string]string{
			"action":      activity.InviteAccepted,
			"user_id":     newUser.ID.String(),
			"invite_id":   inviteID.String(),
			"object_type": "user",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log invite acceptance", zap.Error(logErr))
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.JWTSecret,
		&newUser,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.JWTSecret,
		&newUser,
		string(models.LocalProviderType),
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return models.AuthLoginResponse{}, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	return models.AuthLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s InviteService) CreateInviteChallenge(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.InviteChallengeCreateBody,
) (any, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return nil, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return nil, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	inviteID := ids[0]
	var invite models.Invite
	result := s.DB.Preload("User").Where("id = ?", inviteID).First(&invite)

	if result.RowsAffected == 0 {
		return nil, apierrors.NewAPIError(404, "INVITE_NOT_FOUND")
	}

	if invite.Email != body.Email {
		logger.Warn("Invite email mismatch attempt detected",
			zap.String("invite_id", inviteID.String()),
			zap.String("provided_email", body.Email))
		return nil, apierrors.NewAPIError(404, "INVITE_NOT_FOUND")
	}

	s.DB.Where("invite_id = ? AND type = ?", invite.ID, models.ChallengeTypeInvite).
		Delete(&models.Challenge{})

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
		logger.Error("Failed to create challenge", zap.Error(result.Error))
		return nil, apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
	}

	event := events.NewChallengeUserInvite(
		s.Publisher,
		secret,
		invite.Email,
		invite.User.Email,
		inviteID.String(),
		challenge.ID.String(),
		s.AuthConfig.WebURL,
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
		return models.AuthLoginResponse{}, apierrors.NewAPIError(403, "FORBIDDEN")
	}

	inviteID := ids[0]
	challengeID := ids[1]

	var challenge models.Challenge
	var invite *models.Invite

	// Use transaction with row-level locking to prevent race conditions
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Invite").
			Where("id = ? AND invite_id = ? AND type = ?", challengeID, inviteID, models.ChallengeTypeInvite).
			First(&challenge)

		if result.RowsAffected == 0 {
			return apierrors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
		}

		if challenge.Invite == nil {
			logger.Error("Challenge has no associated invite")
			return apierrors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		invite = challenge.Invite

		if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
			tx.Delete(&challenge)
			return apierrors.NewAPIError(410, "CHALLENGE_EXPIRED")
		}

		if !h.IsDomainAllowed(
			challenge.Invite.Email,
			s.Providers[string(models.LocalProviderType)].Domains,
		) {
			logger.Debug("Domain not allowed")
			return apierrors.NewAPIError(403, "FORBIDDEN")
		}

		match, err := argon2id.ComparePasswordAndHash(
			strings.ToUpper(body.Code),
			challenge.HashedSecret,
		)
		if err != nil || !match {
			return s.handleInviteChallengeFailedAttempt(logger, tx, &challenge, inviteID)
		}

		return nil
	})

	if err != nil {
		return models.AuthLoginResponse{}, err
	}

	return s.createUserFromInvite(logger, invite, &challenge, body.NewPassword, inviteID)
}
