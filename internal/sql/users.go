package sql

import (
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func FindUserByProviderIdentity(
	db *gorm.DB,
	email string,
	providerType models.ProviderType,
	providerKey string,
	preloadMFA bool,
) (models.User, bool, error) {
	tx := db
	if preloadMFA {
		tx = tx.Preload("MFADevices", "is_verified = ?", true)
	}

	var user models.User
	result := tx.Where("email = ? AND provider_type = ? AND provider_key = ?",
		email, providerType, providerKey).
		Find(&user)
	if result.Error != nil {
		return models.User{}, false, result.Error
	}
	return user, result.RowsAffected > 0, nil
}

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
		return processPendingInvites(logger, tx, user)
	})
}

func CreateOrGetUser(
	logger *zap.Logger,
	db *gorm.DB,
	user *models.User,
) (bool, error) {
	var created bool
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(user)
		if result.Error != nil {
			logger.Error("Error creating user", zap.Error(result.Error))
			return result.Error
		}
		if result.RowsAffected > 0 {
			created = true
			return processPendingInvites(logger, tx, user)
		}
		return tx.Where("email = ? AND provider_type = ? AND provider_key = ?",
			user.Email, user.ProviderType, user.ProviderKey).
			First(user).Error
	})
	return created, err
}

func processPendingInvites(logger *zap.Logger, tx *gorm.DB, user *models.User) error {
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
}
