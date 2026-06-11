package services

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/safebucket/safebucket/internal/activity"
	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/handlers"
	h "github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/messaging"
	m "github.com/safebucket/safebucket/internal/middlewares"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"
	"github.com/safebucket/safebucket/internal/sql"
	"github.com/safebucket/safebucket/internal/storage"

	"github.com/alexedwards/argon2id"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InviteService struct {
	DB             *gorm.DB
	Cache          cache.ICache
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
				Post("/validate", s.validateInviteChallengeHandler())
		})
	})

	return r
}

func (s InviteService) validateInviteChallengeHandler() http.HandlerFunc {
	return handlers.AuthFlowHandler(s.AuthConfig.CookieSecureForce, s.ValidateInviteChallenge)
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
			Filter: activity.NewLogFilter(models.ActivityFields{
				Action:      activity.InviteChallengeLocked,
				ChallengeID: challenge.ID.String(),
				InviteID:    inviteID.String(),
				ObjectType:  "challenge",
			}),
		}
		if logErr := s.ActivityLogger.Send(action); logErr != nil {
			logger.Error("Failed to log invite challenge lockout", zap.Error(logErr))
		}

		return apierrors.New(http.StatusForbidden, apierrors.CodeChallengeLocked)
	}

	if updateErr := tx.Save(challenge).Error; updateErr != nil {
		logger.Error("Failed to update attempts counter", zap.Error(updateErr))
		return updateErr
	}

	action := models.Activity{
		Message: activity.InviteChallengeAttemptFailed,
		Object:  nil,
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:       activity.InviteChallengeAttemptFailed,
			ChallengeID:  challenge.ID.String(),
			InviteID:     inviteID.String(),
			AttemptsLeft: strconv.Itoa(challenge.AttemptsLeft),
			ObjectType:   "challenge",
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log failed invite attempt", zap.Error(logErr))
	}

	return apierrors.New(http.StatusUnauthorized, apierrors.CodeWrongCode)
}

func (s InviteService) createUserFromInvite(
	isSecure bool,
	logger *zap.Logger,
	invite *models.Invite,
	challenge *models.Challenge,
	password string,
	inviteID uuid.UUID,
) (handlers.AuthFlowResult, error) {
	newUser := models.User{
		Email:        invite.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
	}

	result := s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected > 0 {
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusConflict, apierrors.CodeUserAlreadyExists)
	}

	hashedPassword, err := h.CreateHash(password)
	if err != nil {
		logger.Error("Failed to hash password", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	newUser.HashedPassword = hashedPassword
	newUser.Role = models.RoleGuest

	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if err = sql.CreateUserWithInvites(logger, tx, &newUser); err != nil {
			logger.Error("Failed to create user with invites", zap.Error(err))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		if deleteResult := tx.Delete(challenge); deleteResult.Error != nil {
			logger.Error("Failed to delete challenge", zap.Error(deleteResult.Error))
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		return nil
	})
	if err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
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
		Filter: activity.NewLogFilter(models.ActivityFields{
			Action:     activity.InviteAccepted,
			UserID:     newUser.ID.String(),
			InviteID:   inviteID.String(),
			ObjectType: rbac.ResourceUser.String(),
		}),
	}
	if logErr := s.ActivityLogger.Send(action); logErr != nil {
		logger.Error("Failed to log invite acceptance", zap.Error(logErr))
	}

	sid := uuid.New().String()
	if sessionErr := cache.CreateSession(s.Cache, newUser.ID.String(), sid); sessionErr != nil {
		logger.Error("Failed to create session", zap.Error(sessionErr))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	accessToken, err := h.NewAccessToken(
		s.AuthConfig.TokenSecret,
		&newUser,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate access token", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	refreshToken, err := h.NewRefreshToken(
		s.AuthConfig.TokenSecret,
		&newUser,
		string(models.LocalProviderType),
		sid,
	)
	if err != nil {
		logger.Error("Failed to generate refresh token", zap.Error(err))
		return handlers.AuthFlowResult{}, apierrors.New(
			http.StatusInternalServerError,
			apierrors.CodeInternalServerError,
		)
	}

	return handlers.AuthFlowResult{
		Status: http.StatusNoContent,
		Body:   nil,
		Cookies: handlers.BuildAuthCookies(
			isSecure,
			accessToken,
			refreshToken,
			string(models.LocalProviderType),
		),
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
		return nil, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return nil, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	if err := enforceEmailIssuanceLimit(
		logger,
		s.Cache,
		string(models.ChallengeTypeInvite),
		body.Email,
		configuration.SecurityInviteMaxPerEmailPerHour,
	); err != nil {
		return nil, err
	}

	inviteID := ids[0]
	var invite models.Invite
	result := s.DB.Preload("User").Where("id = ?", inviteID).First(&invite)

	if result.RowsAffected == 0 {
		return nil, apierrors.New(http.StatusNotFound, apierrors.CodeInviteNotFound)
	}

	if invite.Email != body.Email {
		logger.Warn("Invite email mismatch attempt detected",
			zap.String("invite_id", inviteID.String()),
			zap.String("provided_email", body.Email))
		return nil, apierrors.New(http.StatusNotFound, apierrors.CodeInviteNotFound)
	}

	s.DB.Where("invite_id = ? AND type = ?", invite.ID, models.ChallengeTypeInvite).
		Delete(&models.Challenge{})

	secret, err := h.GenerateSecret()
	if err != nil {
		logger.Error("Failed to generate secret", zap.Error(err))
		return nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

	hashedSecret, err := h.CreateHash(secret)
	if err != nil {
		logger.Error("Failed to hash secret", zap.Error(err))
		return nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
	}

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
		return nil, apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
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

	return nil, nil
}

func (s InviteService) ValidateInviteChallenge(
	isSecure bool,
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
	body models.InviteChallengeValidateBody,
) (handlers.AuthFlowResult, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return handlers.AuthFlowResult{}, apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
	}

	inviteID := ids[0]
	challengeID := ids[1]

	var challenge models.Challenge
	var invite *models.Invite

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Invite").
			Where("id = ? AND invite_id = ? AND type = ?", challengeID, inviteID, models.ChallengeTypeInvite).
			First(&challenge)

		if result.RowsAffected == 0 {
			return apierrors.New(http.StatusNotFound, apierrors.CodeChallengeNotFound)
		}

		if challenge.Invite == nil {
			logger.Error("Challenge has no associated invite")
			return apierrors.New(http.StatusInternalServerError, apierrors.CodeInternalServerError)
		}

		invite = challenge.Invite

		if challenge.ExpiresAt != nil && time.Now().After(*challenge.ExpiresAt) {
			tx.Delete(&challenge)
			return apierrors.New(http.StatusGone, apierrors.CodeChallengeExpired)
		}

		if !h.IsDomainAllowed(
			challenge.Invite.Email,
			s.Providers[string(models.LocalProviderType)].Domains,
		) {
			logger.Debug("Domain not allowed")
			return apierrors.New(http.StatusForbidden, apierrors.CodeForbidden)
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
		return handlers.AuthFlowResult{}, err
	}

	return s.createUserFromInvite(isSecure, logger, invite, &challenge, body.NewPassword, inviteID)
}
