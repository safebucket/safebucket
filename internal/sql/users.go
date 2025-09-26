package sql

import (
	"api/internal/helpers"
	"api/internal/models"
	"api/internal/rbac/groups"

	"github.com/casbin/casbin/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func createUserWithRoleBase(
	tx *gorm.DB,
	enforcer *casbin.Enforcer,
	user *models.User,
	roleFunc func(*casbin.Enforcer, models.User) error,
) error {
	res := tx.Create(&user)
	if res.Error != nil {
		zap.L().Error("Error creating user", zap.Error(res.Error))
		return res.Error
	}

	err := roleFunc(enforcer, *user)
	if err != nil {
		zap.L().Error("can not add user to role", zap.Error(err))
		return err
	}

	err = helpers.AllowUserToSelfModify(enforcer, *user)
	if err != nil {
		zap.L().Error("can not allow user self modify", zap.Error(err))
		return err
	}

	return nil
}

func CreateUserWithRoleAndInvites(
	logger *zap.Logger,
	db *gorm.DB,
	enforcer *casbin.Enforcer,
	user *models.User,
	roleFunc func(*casbin.Enforcer, models.User) error,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := createUserWithRoleBase(tx, enforcer, user, roleFunc)
		if err != nil {
			return err
		}

		var invites []models.Invite
		result := tx.Preload("Bucket").Where("email = ?", user.Email).Find(&invites)
		if result.Error != nil {
			logger.Error("Failed to fetch user invites", zap.Error(result.Error))
			return result.Error
		}

		// Process all invites within the transaction
		for _, invite := range invites {
			var err error
			switch invite.Group {
			case "viewer":
				err = groups.AddUserToViewers(enforcer, invite.Bucket, user.ID.String())
			case "contributor":
				err = groups.AddUserToContributors(enforcer, invite.Bucket, user.ID.String())
			case "owner":
				err = groups.AddUserToOwners(enforcer, invite.Bucket, user.ID.String())
			default:
				logger.Error("Invalid group in invite", zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
				continue
			}

			if err != nil {
				logger.Error("Failed to add user to group", zap.Error(err), zap.String("group", invite.Group), zap.String("bucket_id", invite.BucketID.String()), zap.String("user_id", invite.CreatedBy.String()))
				return err
			}

			// Delete invite within transaction
			deleteResult := tx.Delete(&invite)
			if deleteResult.Error != nil {
				logger.Error("Failed to delete invite", zap.Error(deleteResult.Error), zap.String("invite_id", invite.ID.String()))
				return deleteResult.Error
			}
		}

		return nil
	})
}
