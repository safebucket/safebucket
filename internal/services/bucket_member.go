package services

import (
	"api/internal/activity"
	"api/internal/configuration"
	"api/internal/errors"
	"api/internal/events"
	"api/internal/handlers"
	"api/internal/helpers"
	"api/internal/messaging"
	m "api/internal/middlewares"
	"api/internal/models"
	"api/internal/rbac"
	"api/internal/rbac/groups"
	"api/internal/sql"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketMemberService struct {
	DB             *gorm.DB
	Enforcer       *casbin.Enforcer
	Providers      configuration.Providers
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s BucketMemberService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
		Get("/", handlers.GetListHandler(s.GetBucketMembers))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionGrant, 0)).
		With(m.Validate[models.UpdateMembersBody]).
		Put("/", handlers.UpdateHandler(s.UpdateBucketMembers))

	return r
}

func (s BucketMemberService) GetBucketMembers(logger *zap.Logger, _ models.UserClaims, ids uuid.UUIDs) []models.BucketMember {
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

			var group string
			if groupName == groups.GetBucketOwnerGroup(bucket) {
				group = "owner"
			} else if groupName == groups.GetBucketContributorGroup(bucket) {
				group = "contributor"
			} else if groupName == groups.GetBucketViewerGroup(bucket) {
				group = "viewer"
			}

			userEmailMap[dbUser.Email] = dbUser

			members = append(members, models.BucketMember{
				UserID:    dbUser.ID,
				Email:     dbUser.Email,
				FirstName: dbUser.FirstName,
				LastName:  dbUser.LastName,
				Group:     group,
				Status:    "active",
			})
		}
	}

	var invites []models.Invite
	result = s.DB.Where("bucket_id = ?", bucket.ID).Find(&invites)
	if result.Error != nil {
		logger.Error("Failed to fetch invites", zap.Error(result.Error))
	} else {
		for _, invite := range invites {
			if _, exists := userEmailMap[invite.Email]; exists {
				continue
			}

			members = append(members, models.BucketMember{
				Email:  invite.Email,
				Group:  invite.Group,
				Status: "invited",
			})
		}
	}

	return members
}

func (s BucketMemberService) UpdateBucketMembers(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.UpdateMembersBody,
) (interface{}, error) {
	bucketId := ids[0]

	var providerCfg configuration.Provider
	var ok bool

	providerCfg, ok = s.Providers[user.Provider]
	if !ok {
		return nil, errors.NewAPIError(400, "UNKNOWN_USER_PROVIDER")
	}
	if !providerCfg.SharingOptions.Allowed {
		return nil, errors.NewAPIError(403, "SHARING_DISABLED_FOR_PROVIDER")
	}

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketId).First(&bucket)

	if result.RowsAffected == 0 {
		return nil, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	members := s.GetBucketMembers(logger, user, ids)
	currentMembers := map[string]models.BucketMember{}
	for _, member := range members {
		// This condition ensures there's always at least one owner on a bucket
		if member.Email != user.Email {
			currentMembers[member.Email] = member
		}
	}

	updatedMembers := map[string]models.BucketMemberBody{}
	for _, member := range body.Members {
		// This condition ensures there's always at least one owner on a bucket
		// Or ensures a fallback for the platform admin
		if member.Email != user.Email || len(members) == 0 {
			updatedMembers[member.Email] = member
		}
	}

	changes := s.compareMemberships(currentMembers, updatedMembers)

	for _, member := range changes.ToAdd {
		if helpers.IsDomainAllowed(member.Email, providerCfg.SharingOptions.Domains) {
			s.addMember(logger, user, bucket, member)
		}
	}

	for _, member := range changes.ToUpdate {
		if helpers.IsDomainAllowed(member.Email, providerCfg.SharingOptions.Domains) {
			s.updateMember(logger, user, bucket, member)
		}
	}

	for _, member := range changes.ToDelete {
		if helpers.IsDomainAllowed(member.Email, providerCfg.SharingOptions.Domains) {
			s.deleteMember(logger, user, bucket, member)
		}
	}

	return nil, nil
}

func (s BucketMemberService) compareMemberships(
	currentMembers map[string]models.BucketMember,
	updatedMembers map[string]models.BucketMemberBody,
) models.MembershipChanges {
	changes := models.MembershipChanges{
		ToAdd:    []models.BucketMemberBody{},
		ToUpdate: []models.BucketMemberToUpdate{},
		ToDelete: []models.BucketMember{},
	}

	for email, updatedMember := range updatedMembers {
		if currentMember, exists := currentMembers[email]; exists {
			if currentMember.Group != updatedMember.Group {
				updated := models.BucketMemberToUpdate{
					BucketMember: currentMember,
					NewGroup:     updatedMember.Group,
				}
				changes.ToUpdate = append(changes.ToUpdate, updated)
			}
		} else {
			changes.ToAdd = append(changes.ToAdd, updatedMember)
		}
	}

	for email := range currentMembers {
		if _, exists := updatedMembers[email]; !exists {
			changes.ToDelete = append(changes.ToDelete, currentMembers[email])
		}
	}

	return changes
}

func (s BucketMemberService) addMember(logger *zap.Logger, user models.UserClaims, bucket models.Bucket, invite models.BucketMemberBody) {
	err := sql.WithCasbinTx(s.DB, s.Enforcer, func(tx *gorm.DB, enforcer *casbin.Enforcer) error {
		var invitee models.User
		result := tx.Where("email = ?", invite.Email).First(&invitee)

		if result.RowsAffected == 0 {
			inviteRecord := models.Invite{
				Email:     invite.Email,
				Group:     invite.Group,
				BucketID:  bucket.ID,
				CreatedBy: user.UserID,
			}

			if err := tx.Create(&inviteRecord).Error; err != nil {
				if strings.Contains(err.Error(), "duplicate key") {
					return err
				}
				logger.Error("Failed to create invite", zap.String("email", invite.Email), zap.Error(err))
				return err
			}

			invitationEvent := events.NewUserInvitation(
				s.Publisher,
				inviteRecord.Email,
				user.Email,
				bucket,
				inviteRecord.Group,
				inviteRecord.ID.String(),
				s.WebUrl,
			)
			invitationEvent.Trigger()
		} else {
			bucketSharedEvent := events.NewBucketSharedWith(
				s.Publisher,
				bucket,
				user.Email,
				invite.Email,
			)
			bucketSharedEvent.Trigger()

			var err error
			switch invite.Group {
			case "viewer":
				err = groups.AddUserToViewers(enforcer, bucket, invitee.ID.String())
			case "contributor":
				err = groups.AddUserToContributors(enforcer, bucket, invitee.ID.String())
			case "owner":
				err = groups.AddUserToOwners(enforcer, bucket, invitee.ID.String())
			default:
				return nil
			}

			if err != nil {
				logger.Error("Failed to add user to Casbin group", zap.Error(err))
				return err
			}
		}

		action := models.Activity{
			Message: activity.BucketMemberCreated,
			Filter: activity.NewLogFilter(map[string]string{
				"action":              rbac.ActionGrant.String(),
				"domain":              configuration.DefaultDomain,
				"object_type":         rbac.ResourceBucket.String(),
				"bucket_id":           bucket.ID.String(),
				"user_id":             user.UserID.String(),
				"bucket_member_email": invite.Email,
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log user invitation activity", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to add member", zap.Error(err))
	}
}

func (s BucketMemberService) updateMember(logger *zap.Logger, user models.UserClaims, bucket models.Bucket, member models.BucketMemberToUpdate) {
	err := sql.WithCasbinTx(s.DB, s.Enforcer, func(tx *gorm.DB, enforcer *casbin.Enforcer) error {
		if member.Status == "invited" {
			updateResult := tx.Model(&models.Invite{}).
				Where("bucket_id = ? AND email = ?", bucket.ID, member.Email).
				Update("group", member.NewGroup)

			if updateResult.Error != nil {
				logger.Error("Failed to update invite role", zap.Error(updateResult.Error))
				return updateResult.Error
			}

			if updateResult.RowsAffected == 0 {
				return nil
			}
		} else {
			userId := member.UserID.String()

			var err error
			switch member.Group {
			case "viewer":
				err = groups.RemoveUserFromViewers(enforcer, bucket, userId)
			case "contributor":
				err = groups.RemoveUserFromContributors(enforcer, bucket, userId)
			case "owner":
				err = groups.RemoveUserFromOwners(enforcer, bucket, userId)
			default:
				return nil
			}

			if err != nil {
				logger.Error("Failed to remove user from old role", zap.Error(err))
				return err
			}

			switch member.NewGroup {
			case "owner":
				err = groups.AddUserToOwners(enforcer, bucket, userId)
			case "contributor":
				err = groups.AddUserToContributors(enforcer, bucket, userId)
			case "viewer":
				err = groups.AddUserToViewers(enforcer, bucket, userId)
			default:
				return nil
			}

			if err != nil {
				logger.Error("Failed to add user to new role", zap.Error(err))
				return err
			}
		}

		action := models.Activity{
			Message: activity.BucketMemberUpdated,
			Filter: activity.NewLogFilter(map[string]string{
				"action":              rbac.ActionGrant.String(),
				"domain":              configuration.DefaultDomain,
				"object_type":         rbac.ResourceBucket.String(),
				"bucket_id":           bucket.ID.String(),
				"user_id":             user.UserID.String(),
				"bucket_member_email": member.Email,
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log user role update activity", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to update member", zap.Error(err))
	}
}

func (s BucketMemberService) deleteMember(logger *zap.Logger, user models.UserClaims, bucket models.Bucket, member models.BucketMember) {
	err := sql.WithCasbinTx(s.DB, s.Enforcer, func(tx *gorm.DB, enforcer *casbin.Enforcer) error {
		if member.Status == "invited" {
			deleteResult := tx.Where(
				"bucket_id = ? AND email = ?", bucket.ID, member.Email,
			).Delete(&models.Invite{})

			if deleteResult.Error != nil {
				logger.Error("Failed to delete invite", zap.Error(deleteResult.Error))
				return deleteResult.Error
			}

			if deleteResult.RowsAffected == 0 {
				return nil
			}
		} else {
			var err error
			userIdStr := member.UserID.String()

			switch member.Group {
			case "owner":
				err = groups.RemoveUserFromOwners(enforcer, bucket, userIdStr)
			case "contributor":
				err = groups.RemoveUserFromContributors(enforcer, bucket, userIdStr)
			case "viewer":
				err = groups.RemoveUserFromViewers(enforcer, bucket, userIdStr)
			default:
				return nil
			}

			if err != nil {
				logger.Error("Failed to remove user from role", zap.Error(err))
				return err
			}
		}

		action := models.Activity{
			Message: activity.BucketMemberDeleted,
			Filter: activity.NewLogFilter(map[string]string{
				"action":              rbac.ActionGrant.String(),
				"domain":              configuration.DefaultDomain,
				"object_type":         rbac.ResourceBucket.String(),
				"bucket_id":           bucket.ID.String(),
				"user_id":             user.UserID.String(),
				"bucket_member_email": member.Email,
			}),
		}

		if err := s.ActivityLogger.Send(action); err != nil {
			logger.Error("Failed to log user removal activity", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("Failed to delete member", zap.Error(err))
	}
}
