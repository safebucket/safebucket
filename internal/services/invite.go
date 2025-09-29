package services

import (
	"api/internal/activity"
	"api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	h "api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac/roles"
	"api/internal/sql"
	"api/internal/storage"
	"strings"

	"github.com/alexedwards/argon2id"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InviteService struct {
	DB             *gorm.DB
	Storage        storage.IStorage
	JWTSecret      string
	Enforcer       *casbin.Enforcer
	Publisher      messaging.IPublisher
	Providers      configuration.Providers
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s InviteService) Routes() chi.Router {
	r := chi.NewRouter()

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Validate[models.InviteChallengeCreateBody]).Post("/challenges", handlers.CreateHandler(s.CreateInviteChallenge))

		r.Route("/challenges/{id1}", func(r chi.Router) {
			r.With(m.Validate[models.InviteChallengeValidateBody]).Post("/validate", handlers.CreateHandler(s.ValidateInviteChallenge))
		})
	})

	return r
}

func (s InviteService) CreateInviteChallenge(logger *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeCreateBody) (interface{}, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	if !h.IsDomainAllowed(body.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	inviteId := ids[0]
	var invite models.Invite
	result := s.DB.Where("id = ?", inviteId).First(&invite)

	if result.RowsAffected == 0 {
		return invite, errors.NewAPIError(404, "INVITE_NOT_FOUND")
	} else if invite.Email != body.Email {
		return invite, errors.NewAPIError(400, "INVITE_EMAIL_MISMATCH")
	} else {
		secret, err := h.GenerateSecret(6)
		if err != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		hashedSecret, err := h.CreateHash(secret)

		if err != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		challenge := models.Challenge{
			InviteID:     invite.ID,
			HashedSecret: hashedSecret,
		}

		result = s.DB.Create(&challenge)
		if result.Error != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		event := events.NewChallengeUserInvite(
			s.Publisher,
			secret,
			invite.Email,
			inviteId.String(),
			challenge.ID.String(),
			s.WebUrl,
		)
		event.Trigger()

		return challenge, nil
	}
}

func (s InviteService) ValidateInviteChallenge(logger *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeValidateBody) (models.AuthLoginResponse, error) {
	if _, ok := s.Providers[string(models.LocalProviderType)]; !ok {
		logger.Debug("Local auth provider not activated in the configuration")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	inviteId := ids[0]
	challengeId := ids[1]

	var challenge models.Challenge

	result := s.DB.Preload("Invite").Where("id = ? AND invite_id = ?", challengeId, inviteId).First(&challenge)

	if !h.IsDomainAllowed(challenge.Invite.Email, s.Providers[string(models.LocalProviderType)].Domains) {
		logger.Debug("Domain not allowed")
		return models.AuthLoginResponse{}, errors.NewAPIError(403, "FORBIDDEN")
	}

	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, errors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
	}

	match, err := argon2id.ComparePasswordAndHash(strings.ToUpper(body.Code), challenge.HashedSecret)
	if err != nil || !match {
		return models.AuthLoginResponse{}, errors.NewAPIError(401, "WRONG_CODE")
	}

	// If the code matches, we create a new user, create policies for the user, and return the access token.
	newUser := models.User{
		Email:        challenge.Invite.Email,
		ProviderType: models.LocalProviderType,
		ProviderKey:  string(models.LocalProviderType),
	}

	result = s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected == 0 {
		err := sql.CreateUserWithRoleAndInvites(logger, s.DB, s.Enforcer, &newUser, roles.AddUserToRoleGuest)

		if err != nil {
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "INTERNAL_SERVER_ERROR")
		}

		// Generate tokens after successful transaction commit
		accessToken, err := h.NewAccessToken(s.JWTSecret, &newUser, string(models.LocalProviderType))
		if err != nil {
			logger.Error("Failed to generate access token", zap.Error(err))
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_ACCESS_TOKEN_FAILED")
		}
		refreshToken, err := h.NewRefreshToken(s.JWTSecret, &newUser, string(models.LocalProviderType))
		if err != nil {
			logger.Error("Failed to generate refresh token", zap.Error(err))
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_REFRESH_TOKEN_FAILED")
		}

		tokens := models.AuthLoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
		return tokens, nil
	} else {
		return models.AuthLoginResponse{}, errors.NewAPIError(401, "USER_ALREADY_EXISTS")
	}
}
