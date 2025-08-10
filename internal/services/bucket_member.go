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
	Publisher      *messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	WebUrl         string
}

func (s BucketMemberService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
		Get("/", handlers.GetListHandler(s.GetBucketMembers))

	r.With(m.Authorize(s.Enforcer, rbac.ResourceBucket, rbac.ActionRead, 0)).
		With(m.Validate[models.UpdateMembersBody]).
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

func (s BucketMemberService) UpdateBucketMembers(
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.UpdateMembersBody,
) (interface{}, error) {
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

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketId).First(&bucket)

	if result.RowsAffected == 0 {
		return nil, errors.NewAPIError(404, "BUCKET_NOT_FOUND")
	}

	members := s.GetBucketMembers(user, ids)
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
		if member.Email != user.Email {
			updatedMembers[member.Email] = member
		}
	}

	changes := s.compareMemberships(currentMembers, updatedMembers)

	for _, member := range changes.ToAdd {
		if s.isDomainAllowed(user, member.Email, providerCfg) {
			s.addMember(user, bucket, member)
		}
	}

	for _, member := range changes.ToUpdate {
		if s.isDomainAllowed(user, member.Email, providerCfg) {
			s.updateMember(user, bucket, member)
		}
	}

	for _, member := range changes.ToDelete {
		if s.isDomainAllowed(user, member.Email, providerCfg) {
			s.deleteMember(user, bucket, member)
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
		ToUpdate: []models.BucketMember{},
		ToDelete: []models.BucketMember{},
	}

	for email, updatedMember := range updatedMembers {
		if currentMember, exists := currentMembers[email]; exists {
			if currentMember.Role != updatedMember.Group {
				updated := models.BucketMember{
					UserID: currentMember.UserID,
					Email:  email,
					Role:   updatedMember.Group,
					Status: currentMember.Status,
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

func (s BucketMemberService) isDomainAllowed(
	user models.UserClaims,
	email string,
	providerCfg configuration.Provider,
) bool {
	domainParts := strings.Split(email, "@")
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

func (s BucketMemberService) addMember(user models.UserClaims, bucket models.Bucket, invite models.BucketMemberBody) {
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
				return
			}
			zap.L().Error("Failed to create invite", zap.String("email", invite.Email), zap.Error(err))
			return
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
		return
	}

	if err != nil {
		return
	}

	action := models.Activity{
		Message: activity.BucketMemberCreated,
		Filter: activity.NewLogFilter(map[string]string{
			"action":              rbac.ActionShare.String(),
			"domain":              configuration.DefaultDomain,
			"object_type":         rbac.ResourceBucket.String(),
			"bucket_id":           bucket.ID.String(),
			"user_id":             user.UserID.String(),
			"bucket_member_email": invite.Email,
		}),
	}

	err = s.ActivityLogger.Send(action)
	if err != nil {
		zap.L().Error("Failed to log user invitation activity", zap.Error(err))
	}
}

func (s BucketMemberService) updateMember(user models.UserClaims, bucket models.Bucket, member models.BucketMember) {
	if member.Status == "invited" {
		updateResult := s.DB.Model(&models.Invite{}).
			Where("bucket_id = ? AND email = ?", bucket.ID, member.Email).
			Update("group", member.Role)

		if updateResult.Error != nil {
			zap.L().Error("Failed to update invite role", zap.Error(updateResult.Error))
			return
		}

		if updateResult.RowsAffected == 0 {
			return
		}
	} else {
		userId := member.UserID.String()

		_ = groups.RemoveUserFromOwners(s.Enforcer, bucket, userId)
		_ = groups.RemoveUserFromContributors(s.Enforcer, bucket, userId)
		_ = groups.RemoveUserFromViewers(s.Enforcer, bucket, userId)

		var err error
		switch member.Role {
		case "owner":
			err = groups.AddUserToOwners(s.Enforcer, bucket, userId)
		case "contributor":
			err = groups.AddUserToContributors(s.Enforcer, bucket, userId)
		case "viewer":
			err = groups.AddUserToViewers(s.Enforcer, bucket, userId)
		default:
			return
		}

		if err != nil {
			zap.L().Error("Failed to add user to new role", zap.Error(err))
			return
		}
	}

	action := models.Activity{
		Message: activity.BucketMemberUpdated,
		Filter: activity.NewLogFilter(map[string]string{
			"action":              rbac.ActionShare.String(),
			"domain":              configuration.DefaultDomain,
			"object_type":         rbac.ResourceBucket.String(),
			"bucket_id":           bucket.ID.String(),
			"user_id":             user.UserID.String(),
			"bucket_member_email": member.Email,
		}),
	}

	err := s.ActivityLogger.Send(action)
	if err != nil {
		zap.L().Error("Failed to log user role update activity", zap.Error(err))
	}
}

func (s BucketMemberService) deleteMember(user models.UserClaims, bucket models.Bucket, member models.BucketMember) {
	if member.Status == "invited" {
		deleteResult := s.DB.Where(
			"bucket_id = ? AND email = ?", bucket.ID, member.Email,
		).Delete(&models.Invite{})

		if deleteResult.Error != nil {
			zap.L().Error("Failed to delete invite", zap.Error(deleteResult.Error))
			return
		}

		if deleteResult.RowsAffected == 0 {
			return
		}
	} else {
		// Delete user from Casbin groups (active user)
		var err error
		userIdStr := member.UserID.String()

		switch member.Role {
		case "owner":
			err = groups.RemoveUserFromOwners(s.Enforcer, bucket, userIdStr)
		case "contributor":
			err = groups.RemoveUserFromContributors(s.Enforcer, bucket, userIdStr)
		case "viewer":
			err = groups.RemoveUserFromViewers(s.Enforcer, bucket, userIdStr)
		default:
			return
		}

		if err != nil {
			zap.L().Error("Failed to remove user from role", zap.Error(err))
			return
		}
	}

	action := models.Activity{
		Message: activity.BucketMemberDeleted,
		Filter: activity.NewLogFilter(map[string]string{
			"action":              rbac.ActionShare.String(),
			"domain":              configuration.DefaultDomain,
			"object_type":         rbac.ResourceBucket.String(),
			"bucket_id":           bucket.ID.String(),
			"user_id":             user.UserID.String(),
			"bucket_member_email": member.Email,
		}),
	}

	err := s.ActivityLogger.Send(action)
	if err != nil {
		zap.L().Error("Failed to log user removal activity", zap.Error(err))
	}
}
