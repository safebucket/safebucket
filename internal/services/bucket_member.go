package services

import (
	"api/internal/activity"
	"api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/groups"
	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type BucketMemberService struct {
	DB             *gorm.DB
	Enforcer       *casbin.Enforcer
	Providers      configuration.Providers
	Publisher      *messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s BucketMemberService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
		Get("/", handlers.GetListHandler(s.GetBucketMembers))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionShare, 0)).
		Put("/", handlers.UpdateHandler(s.UpdateBucketMembers))

	return r
}

func (s BucketMemberService) GetBucketMembers(_ models.UserClaims, ids uuid.UUIDs) []models.BucketMember {
	bucketId := ids[0]
	var bucket models.Bucket

	result := s.DB.Where("id = ?", bucketId).First(&bucket)
	if result.RowsAffected == 0 {
		return []models.BucketMember{}
	}

	var members []models.BucketMember
	userEmailMap := make(map[string]models.User)

	owners, err := s.Enforcer.GetFilteredGroupingPolicy(0, "", groups.GetBucketOwnerGroup(bucket), configuration.DefaultDomain)
	if err != nil {
		return []models.BucketMember{}
	}

	contributors, err := s.Enforcer.GetFilteredGroupingPolicy(0, "", groups.GetBucketContributorGroup(bucket), configuration.DefaultDomain)
	if err != nil {
		return []models.BucketMember{}
	}

	viewers, err := s.Enforcer.GetFilteredGroupingPolicy(0, "", groups.GetBucketViewerGroup(bucket), configuration.DefaultDomain)
	if err != nil {
		return []models.BucketMember{}
	}

	var allPolicies [][]string
	allPolicies = append(allPolicies, owners...)
	allPolicies = append(allPolicies, contributors...)
	allPolicies = append(allPolicies, viewers...)

	for _, policy := range allPolicies {
		if !strings.HasPrefix(policy[0], "group") {
			userId := policy[0]
			groupName := policy[1]

			var dbUser models.User
			result := s.DB.Where("id = ?", userId).First(&dbUser)
			if result.Error != nil {
				continue
			}

			var role string
			if strings.Contains(groupName, rbac.GroupOwner.String()) {
				role = "owner"
			} else if strings.Contains(groupName, rbac.GroupContributor.String()) {
				role = "contributor"
			} else if strings.Contains(groupName, rbac.GroupViewer.String()) {
				role = "viewer"
			}

			userEmailMap[dbUser.Email] = dbUser

			members = append(members, models.BucketMember{
				UserID:    dbUser.ID,
				Email:     dbUser.Email,
				FirstName: dbUser.FirstName,
				LastName:  dbUser.LastName,
				Role:      role,
				Status:    "active",
			})
		}
	}

	var invites []models.Invite
	result = s.DB.Where("bucket_id = ?", bucket.ID).Find(&invites)
	if result.Error != nil {
		zap.L().Error("Failed to fetch invites", zap.Error(result.Error))
	} else {
		for _, invite := range invites {
			if _, exists := userEmailMap[invite.Email]; exists {
				continue
			}

			members = append(members, models.BucketMember{
				Email:  invite.Email,
				Role:   invite.Group,
				Status: "invited",
			})
		}
	}

	return members
}

func (s BucketMemberService) UpdateBucketMembers(user models.UserClaims, ids uuid.UUIDs, body models.InviteBody) ([]models.InviteResult, error) {
	bucketId := ids[0]

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
	}

	var results []models.InviteResult

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketId).First(&bucket)

	if result.RowsAffected == 0 {
		return nil, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	for _, invite := range body.Invites {
		result := s.processInvite(user, bucket, invite, providerCfg)
		results = append(results, result)
	}

	return results, nil
}

func (s BucketMemberService) isDomainAllowed(
	user models.UserClaims,
	invite models.BucketInvitee,
	providerCfg configuration.Provider,
) bool {
	domainParts := strings.Split(invite.Email, "@")
	if len(domainParts) != 2 {
		return false
	}

	emailDomain := domainParts[1]

	// TODO: migrate local authent to a provider
	if user.Provider == configuration.AuthLocalProviderName {
		return true // Local users are allowed to invite anyone
	} else {
		for _, domain := range providerCfg.SharingOptions.AllowedDomains {
			if strings.EqualFold(emailDomain, domain) {
				return true
			}
		}
	}

	return false
}

func (s BucketMemberService) processInvite(
	user models.UserClaims,
	bucket models.Bucket,
	invite models.BucketInvitee,
	providerCfg configuration.Provider,
) models.InviteResult {
	inviteResult := models.InviteResult{
		Email: invite.Email,
		Group: invite.Group,
	}

	if !s.isDomainAllowed(user, invite, providerCfg) {
		inviteResult.Status = "domain_not_allowed"
		return inviteResult
	}

	var invitee models.User
	result := s.DB.Where("email = ?", invite.Email).First(&invitee)

	if result.RowsAffected == 0 {
		// User is not found in database - create invitation record
		invite := models.Invite{
			Email:     invite.Email,
			Group:     invite.Group,
			BucketID:  bucket.ID,
			CreatedBy: user.UserID,
		}

		if err := s.DB.Create(&invite).Error; err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				inviteResult.Status = "invite_already_exists"
				return inviteResult
			}
			inviteResult.Status = "create_invite_failed"
			return inviteResult
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

	var err error
	switch invite.Group {
	case "viewer":
		err = groups.AddUserToViewers(s.Enforcer, bucket, invitee.ID.String())
	case "contributor":
		err = groups.AddUserToContributors(s.Enforcer, bucket, invitee.ID.String())
	case "owner":
		err = groups.AddUserToOwners(s.Enforcer, bucket, invitee.ID.String())
	default:
		inviteResult.Status = "invalid_group"
		return inviteResult
	}

	if err != nil {
		inviteResult.Status = "add_to_group_failed"
		return inviteResult
	}

	inviteResult.Status = "success"

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

	return inviteResult
}
