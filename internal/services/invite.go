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

func (s InviteService) CreateInviteChallenge(_ *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeCreateBody) (interface{}, error) {
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
			s.Publisher,
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

func (s InviteService) ValidateInviteChallenge(logger *zap.Logger, _ models.UserClaims, ids uuid.UUIDs, body models.InviteChallengeValidateBody) (models.AuthLoginResponse, error) {
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

	result = s.DB.Where("email = ?", newUser.Email).First(&newUser)
	if result.RowsAffected == 0 {
		// Start transaction for user creation and invite processing
		tx := s.DB.Begin()
		if tx.Error != nil {
			logger.Error("Failed to start transaction", zap.Error(tx.Error))
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "TRANSACTION_START_FAILED")
		}

		err = sql.CreateUserWithRoleBase(tx, s.Enforcer, &newUser, roles.AddUserToRoleGuest)
		if err != nil {
			logger.Error("Failed to create user with role", zap.Error(err))
			tx.Rollback()
			return models.AuthLoginResponse{}, err
		}

		var invites []models.Invite
		result = tx.Preload("Bucket").Where("email = ?", challenge.Invite.Email).Find(&invites)
		if result.Error != nil {
			logger.Error("Failed to fetch user invites", zap.Error(result.Error))
			tx.Rollback()
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "FETCH_INVITES_FAILED")
		}

		// Process all invites within the transaction
		for _, invite := range invites {
			var err error
			switch invite.Group {
			case "viewer":
				err = groups.AddUserToViewers(s.Enforcer, invite.Bucket, newUser.ID.String())
			case "contributor":
				err = groups.AddUserToContributors(s.Enforcer, invite.Bucket, newUser.ID.String())
			case "owner":
				err = groups.AddUserToOwners(s.Enforcer, invite.Bucket, newUser.ID.String())
			default:
				logger.Error("Invalid group in invite", zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
				continue
			}

			if err != nil {
				logger.Error("Failed to add user to group", zap.Error(err), zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
				tx.Rollback()
				return models.AuthLoginResponse{}, errors.NewAPIError(500, "ROLE_ASSIGNMENT_FAILED")
			}

			// Delete invite within transaction
			deleteResult := tx.Delete(&invite)
			if deleteResult.Error != nil {
				logger.Error("Failed to delete invite", zap.Error(deleteResult.Error), zap.String("invite_id", invite.ID.String()))
				tx.Rollback()
				return models.AuthLoginResponse{}, errors.NewAPIError(500, "INVITE_CLEANUP_FAILED")
			}
		}

		if err := tx.Commit().Error; err != nil {
			logger.Error("Failed to commit transaction", zap.Error(err))
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "TRANSACTION_COMMIT_FAILED")
		}

		// Generate tokens after successful transaction commit
		accessToken, err := h.NewAccessToken(s.JWTSecret, &newUser, configuration.LocalAuthProviderType)
		if err != nil {
			logger.Error("Failed to generate access token", zap.Error(err))
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_ACCESS_TOKEN_FAILED")
		}
		refreshToken, err := h.NewRefreshToken(s.JWTSecret, &newUser, configuration.LocalAuthProviderType)
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
