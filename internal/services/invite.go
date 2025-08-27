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
	"api/internal/rbac/groups"
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
	Publisher      *messaging.IPublisher
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

func (s InviteService) CreateInviteChallenge(_ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeCreateBody) (interface{}, error) {
	inviteId := ids[0]
	var invite models.Invite
	result := s.DB.Where("id = ?", inviteId).First(&invite)

	if result.RowsAffected == 0 {
		return invite, errors.NewAPIError(404, "INVITE_NOT_FOUND")
	} else if invite.Email != body.Email {
		return invite, errors.NewAPIError(400, "INVITE_EMAIL_MISMATCH") //Todo: In the frontend, "an email has been sent if the email is linked to this invitation".
	} else {
		secret, err := h.GenerateSecret(6)
		if err != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		hashedSecret, err := h.CreateHash(secret)

		if err != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		// Create a new challenge for the invite
		challenge := models.Challenge{
			InviteID:     invite.ID,
			HashedSecret: hashedSecret,
		}

		result = s.DB.Create(&challenge)
		if result.Error != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		event := events.NewChallengeUserInvite(
			*s.Publisher,
			secret,
			invite.Email,
			inviteId.String(),
			challenge.ID.String(),
			s.WebUrl,
		)
		event.Trigger()

		return challenge, nil // TODO: In the frontend, "an email has been sent if the email is linked to this invitation".

	}
}

func (s InviteService) ValidateInviteChallenge(_ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeValidateBody) (models.AuthLoginResponse, error) {
	inviteId := ids[0]
	challengeId := ids[1]

	var challenge models.Challenge

	result := s.DB.Preload("Invite").Where("id = ? AND invite_id = ?", challengeId, inviteId).First(&challenge)

	if result.RowsAffected == 0 {
		return models.AuthLoginResponse{}, errors.NewAPIError(404, "CHALLENGE_NOT_FOUND")
	}

	match, err := argon2id.ComparePasswordAndHash(strings.ToUpper(body.Code), challenge.HashedSecret)
	if err != nil || !match {
		return models.AuthLoginResponse{}, errors.NewAPIError(401, "WRONG_CODE")
	}

	// If the code matches, we create a new user, create policies for the user, and return the access token.
	newUser := models.User{
		Email: challenge.Invite.Email,
	}

	zap.L().Info("Creating new user", zap.Any("user", newUser))

	result = s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected == 0 {
		err = sql.CreateUserWithRole(s.DB, s.Enforcer, &newUser, roles.AddUserToRoleGuest)
		if err != nil {
			return models.AuthLoginResponse{}, err
		}

		var invites []models.Invite

		s.DB.Preload("Bucket").Where("email = ?", challenge.Invite.Email).Find(&invites)

		for _, invite := range invites {
			zap.L().Info("User inserted", zap.Any("user", newUser))
			var err error
			switch invite.Group {
			case "viewer":
				err = groups.AddUserToViewers(s.Enforcer, invite.Bucket, newUser.ID.String())
			case "contributor":
				err = groups.AddUserToContributors(s.Enforcer, invite.Bucket, newUser.ID.String())
			case "owner":
				err = groups.AddUserToOwners(s.Enforcer, invite.Bucket, newUser.ID.String())
			default:
				zap.L().Error("Invalid group in invite", zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
			}
			if err != nil {
				zap.L().Error("Failed to add user to group", zap.Error(err), zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
			}
			s.DB.Delete(&invite)
		}

		accessToken, err := h.NewAccessToken(s.JWTSecret, &newUser, configuration.LocalAuthProviderType)
		if err != nil {
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_ACCESS_TOKEN_FAILED")
		}
		refreshToken, err := h.NewRefreshToken(s.JWTSecret, &newUser, configuration.LocalAuthProviderType)
		if err != nil {
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_REFRESH_TOKEN_FAILED")
		}

		tokens := models.AuthLoginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}
		return tokens, err

	} else {
		return models.AuthLoginResponse{}, errors.NewAPIError(401, "USER_ALREADY_EXISTS")
	}
}
