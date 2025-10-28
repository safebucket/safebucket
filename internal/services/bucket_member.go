package services

import (
	"strings"

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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BucketMemberService struct {
	DB             *gorm.DB
	Providers      configuration.Providers
	Publisher      messaging.IPublisher
	ActivityLogger activity.IActivityLogger
	WebURL         string
}

func (s BucketMemberService) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(m.AuthorizeGroup(s.DB, models.GroupViewer, 0)).
		Get("/", handlers.GetListHandler(s.GetBucketMembers))

	r.With(m.AuthorizeGroup(s.DB, models.GroupOwner, 0)).
		With(m.Validate[models.UpdateMembersBody]).
		Put("/", handlers.UpdateHandler(s.UpdateBucketMembers))

	return r
}

func (s BucketMemberService) GetBucketMembers(
	logger *zap.Logger,
	_ models.UserClaims,
	ids uuid.UUIDs,
) []models.BucketMember {
	bucketID := ids[0]
	var bucket models.Bucket

	result := s.DB.Where("id = ?", bucketID).First(&bucket)
	if result.RowsAffected == 0 {
		return []models.BucketMember{}
	}

	var membersList []models.BucketMember
	userEmailMap := make(map[string]models.User)

	memberships, err := rbac.GetBucketMembers(s.DB, bucketID)
	if err != nil {
		logger.Error("Failed to fetch bucket memberships", zap.Error(err))
		return []models.BucketMember{}
	}

	for _, membership := range memberships {
		userEmailMap[membership.User.Email] = membership.User

		membersList = append(membersList, models.BucketMember{
			UserID:    membership.User.ID,
			Email:     membership.User.Email,
			FirstName: membership.User.FirstName,
			LastName:  membership.User.LastName,
			Group:     membership.Group,
			Status:    "active",
		})
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

			membersList = append(membersList, models.BucketMember{
				Email:  invite.Email,
				Group:  invite.Group,
				Status: "invited",
			})
		}
	}

	return membersList
}

func (s BucketMemberService) UpdateBucketMembers(
	logger *zap.Logger,
	user models.UserClaims,
	ids uuid.UUIDs,
	body models.UpdateMembersBody,
) error {
	bucketID := ids[0]

	var providerCfg configuration.Provider
	var ok bool

	providerCfg, ok = s.Providers[user.Provider]
	if !ok {
		return errors.NewAPIError(400, "UNKNOWN_USER_PROVIDER")
	}
	if !providerCfg.SharingOptions.Allowed {
		return errors.NewAPIError(403, "SHARING_DISABLED_FOR_PROVIDER")
	}

	var bucket models.Bucket
	result := s.DB.Where("id = ?", bucketID).First(&bucket)

	if result.RowsAffected == 0 {
		return errors.NewAPIError(404, "BUCKET_NOT_FOUND")
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

	return nil
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

func (s BucketMemberService) addMember(
	logger *zap.Logger,
	user models.UserClaims,
	bucket models.Bucket,
	invite models.BucketMemberBody,
) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		var invitee models.User
		result := tx.Where("email = ?", invite.Email).First(&invitee)

		if result.RowsAffected == 0 {
			// User doesn't exist yet - create an invite
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
				logger.Error(
					"Failed to create invite",
					zap.String("email", invite.Email),
					zap.Error(err),
				)
				return err
			}

			invitationEvent := events.NewUserInvitation(
				s.Publisher,
				inviteRecord.Email,
				user.Email,
				bucket,
				inviteRecord.Group,
				inviteRecord.ID.String(),
				s.WebURL,
			)
			invitationEvent.Trigger()
		} else {
			// User exists - create membership directly
			bucketSharedEvent := events.NewBucketSharedWith(
				s.Publisher,
				bucket,
				user.Email,
				invite.Email,
			)
			bucketSharedEvent.Trigger()

			err := rbac.CreateMembership(tx, invitee.ID, bucket.ID, invite.Group)
			if err != nil {
				logger.Error("Failed to create membership", zap.Error(err))
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

func (s BucketMemberService) updateMember(
	logger *zap.Logger,
	user models.UserClaims,
	bucket models.Bucket,
	member models.BucketMemberToUpdate,
) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
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
			err := rbac.UpdateMembership(tx, member.UserID, bucket.ID, member.NewGroup)
			if err != nil {
				logger.Error("Failed to update membership", zap.Error(err))
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

func (s BucketMemberService) deleteMember(
	logger *zap.Logger,
	user models.UserClaims,
	bucket models.Bucket,
	member models.BucketMember,
) {
	err := s.DB.Transaction(func(tx *gorm.DB) error {
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
			err := rbac.DeleteMembership(tx, member.UserID, bucket.ID)
			if err != nil {
				logger.Error("Failed to delete membership", zap.Error(err))
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
