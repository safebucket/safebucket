package sql

import (
	"api/internal/models"
	"api/internal/rbac"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CreateUserWithInvites creates a user and processes any pending invites
// Converting them to memberships in a single transaction.
func CreateUserWithInvites(
	logger *zap.Logger,
	db *gorm.DB,
	user *models.User,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			logger.Error("Error creating user", zap.Error(err))
			return err
		}

		var invites []models.Invite
		if err := tx.Preload("Bucket").Where("email = ?", user.Email).Find(&invites).Error; err != nil {
			logger.Error("Failed to fetch user invites", zap.Error(err))
			return err
		}

		for _, invite := range invites {
			if err := rbac.CreateMembership(tx, user.ID, invite.BucketID, invite.Group); err != nil {
				logger.Error("Failed to create membership from invite", zap.Error(err),
					zap.String("group", string(invite.Group)),
					zap.String("bucket_id", invite.BucketID.String()))
				return err
			}

			if err := tx.Delete(&invite).Error; err != nil {
				logger.Error("Failed to delete processed invite", zap.Error(err),
					zap.String("invite_id", invite.ID.String()))
				return err
			}
		}

		return nil
	})
}
