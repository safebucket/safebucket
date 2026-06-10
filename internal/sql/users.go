package sql

import (
	"errors"
	"net/http"

	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/rbac"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetUserByID(db *gorm.DB, userID uuid.UUID) (models.User, error) {
	var user models.User

	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, apierrors.New(http.StatusNotFound, apierrors.CodeUserNotFound)
		}
		return models.User{}, err
	}

	return user, nil
}

func FindUserByIdentityProvider(
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
