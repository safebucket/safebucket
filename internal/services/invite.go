package services

import (
	"api/internal/activity"
	"api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/helpers"
	h "api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/groups"
	"api/internal/rbac/roles"
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
	JWTConf        models.JWTConfiguration
	Enforcer       *casbin.Enforcer
	Publisher      *messaging.IPublisher
	Providers      configuration.Providers
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s InviteService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Validate[models.InviteBody]).Post("/", handlers.CreateHandler(s.CreateInvite))

	r.Route("/{id0}", func(r chi.Router) {
		r.With(m.Validate[models.InviteChallengeCreateBody]).Post("/challenges", handlers.CreateHandler(s.CreateInviteChallenge))

		r.Route("/challenges/{id1}", func(r chi.Router) {
			r.With(m.Validate[models.InviteChallengeValidateBody]).Post("/validate", handlers.CreateHandler(s.ValidateInviteChallenge))
		})
	})

	return r
}

func (s InviteService) CreateInvite(user models.UserClaims, ids uuid.UUIDs, body models.InviteBody) ([]models.InviteResult, error) {
	if user.UserID == uuid.Nil {
		return nil, errors.NewAPIError(401, "INVALID_USER")
	}

	bucketId := body.BucketID

	// Check if the user is allowed to invite others to this bucket
	authorized, err := s.Enforcer.Enforce(configuration.DefaultDomain,
		user.UserID.String(),
		rbac.ResourceBucket.String(),
		bucketId.String(),
		rbac.ActionShare.String())

	if err != nil {
		return nil, errors.NewAPIError(500, "INTERNAL_ERROR")
	}
	if !authorized {
		return nil, errors.NewAPIError(403, "NOT_AUTHORIZED_TO_INVITE")
	}

	// TODO: Validate that if the OPENID configuration for the user is set, the user is allowed to invite others
	var providerCfg configuration.Provider
	var ok bool

	if user.Provider != configuration.AuthLocalProviderName {
		providerCfg, ok = s.Providers[user.Provider]
		if !ok {
			return nil, errors.NewAPIError(400, "UNKNOWN_USER_PROVIDER")
		}

		if !providerCfg.SharingOptions.Enabled {
			return nil, errors.NewAPIError(403, "SHARING_DISABLED_FOR_PROVIDER")
		}

		//todo: validate if the user is allowed to invite this domain
	}

	var results []models.InviteResult

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketId).First(&bucket)

	if result.RowsAffected == 0 {
		return nil, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	for _, invite := range body.Invites {
		inviteResult := models.InviteResult{
			Email: invite.Email,
			Group: invite.Group,
		}

		domainParts := strings.Split(inviteResult.Email, "@")
		if len(domainParts) != 2 {
			return nil, errors.NewAPIError(400, "INVALID_USER_EMAIL")
		}

		emailDomain := domainParts[1]

		allowed := false

		// TODO: migrate local authent to a provider
		if user.Provider == configuration.AuthLocalProviderName {
			allowed = true // Local users are allowed to invite anyone
		} else {
			for _, domain := range providerCfg.SharingOptions.AllowedDomains {
				if strings.EqualFold(emailDomain, domain) {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			inviteResult.Status = "domain_not_allowed"
			results = append(results, inviteResult)
			continue
		}

		var invitee models.User
		result = s.DB.Where("email = ?", invite.Email).First(&invitee)

		if result.RowsAffected == 0 {
			// User not found in database - create invitation record
			invite := models.Invite{
				Email:     invite.Email,
				Group:     invite.Group,
				BucketID:  bucket.ID,
				CreatedBy: user.UserID,
			}

			if err := s.DB.Create(&invite).Error; err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					inviteResult.Status = "invite_already_exists"
					results = append(results, inviteResult)
					continue
				}
				inviteResult.Status = "create_invite_failed"
				results = append(results, inviteResult)
				continue
			}

			// Send invitation email to new user
			invitationEvent := events.NewUserInvitation(
				*s.Publisher,
				invite.Email,
				user.Email,
				bucket,
				invite.Group,
				invite.ID.String(),
				s.WebUrl,
			)
			invitationEvent.Trigger()
		} else {
			// User already exists - send bucket shared notification
			bucketSharedEvent := events.NewBucketSharedWith(
				*s.Publisher,
				bucket,
				user.Email,
				invite.Email,
			)
			bucketSharedEvent.Trigger()
		}

		switch invite.Group {
		case "viewer":
			err = groups.AddUserToViewers(s.Enforcer, bucket, invitee.ID.String())
		case "contributor":
			err = groups.AddUserToContributors(s.Enforcer, bucket, invitee.ID.String())
		case "owner":
			err = groups.AddUserToOwners(s.Enforcer, bucket, invitee.ID.String())
		default:
			inviteResult.Status = "invalid_group"
			results = append(results, inviteResult)
			continue
		}
		if err != nil {
			inviteResult.Status = "add_to_group_failed"
			results = append(results, inviteResult)
			continue
		}

		inviteResult.Status = "success"
		results = append(results, inviteResult)

		// Log user invitation activity
		action := models.Activity{
			Message: activity.UserInvited,
			Filter: activity.NewLogFilter(map[string]string{
				"action":        rbac.ActionShare.String(),
				"domain":        configuration.DefaultDomain,
				"object_type":   rbac.ResourceBucket.String(),
				"bucket_id":     bucket.ID.String(),
				"user_id":       user.UserID.String(),
				"invited_email": invite.Email,
				"invited_group": invite.Group,
			}),
		}
		err = s.ActivityLogger.Send(action)
		if err != nil {
			zap.L().Error("Failed to log user invitation activity", zap.Error(err))
		}

	}

	return results, nil // TODO: Normalize messages ?
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
		secret, err := helpers.GenerateSecret(6)
		if err != nil {
			return invite, errors.NewAPIError(500, "INVITE_CHALLENGE_CREATION_FAILED")
		}

		hashedSecret, err := helpers.CreateHash(secret)

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
		// Create a new user
		s.DB.Create(&newUser) // No password is set, as this is an invite
		zap.L().Info("User inserted", zap.Any("user", newUser))
		err = roles.AddUserToRoleGuest(s.Enforcer, newUser)
		if err != nil {
			zap.L().Error("can not add user to role user", zap.Error(err))
		}
		err = roles.AllowUserToSelfModify(s.Enforcer, newUser) // Allow the user to modify their own data (reset password, etc.)
		if err != nil {
			zap.L().Error("can not allow user to self modify", zap.Error(err))
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

		accessToken, err := h.NewAccessToken(s.JWTConf.Secret, &newUser, configuration.AuthLocalProviderName)
		if err != nil {
			return models.AuthLoginResponse{}, errors.NewAPIError(500, "GENERATE_ACCESS_TOKEN_FAILED")
		}
		refreshToken, err := h.NewRefreshToken(s.JWTConf.Secret, &newUser, configuration.AuthLocalProviderName)
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
