package rbac

import (
	"errors"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetUserMembership(
	db *gorm.DB,
	userID uuid.UUID,
	bucketID uuid.UUID,
) (*models.Membership, error) {
	var membership models.Membership
	err := db.Where("user_id = ? AND bucket_id = ?", userID, bucketID).First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func GetBucketMembers(db *gorm.DB, bucketID uuid.UUID) ([]models.Membership, error) {
	var memberships []models.Membership
	err := db.Where("bucket_id = ?", bucketID).Preload("User").Find(&memberships).Error
	return memberships, err
}

func GetUserBuckets(db *gorm.DB, userID uuid.UUID) ([]models.Membership, error) {
	var memberships []models.Membership
	err := db.Where("user_id = ?", userID).Preload("Bucket").Find(&memberships).Error
	return memberships, err
}

func CreateMembership(db *gorm.DB, userID uuid.UUID, bucketID uuid.UUID, group models.Group) error {
	membership := models.Membership{
		UserID:   userID,
		BucketID: bucketID,
		Group:    group,
	}
	return db.Create(&membership).Error
}

func UpdateMembership(
	db *gorm.DB,
	userID uuid.UUID,
	bucketID uuid.UUID,
	newGroup models.Group,
) error {
	return db.Model(&models.Membership{}).
		Where("user_id = ? AND bucket_id = ?", userID, bucketID).
		Update("group", newGroup).Error
}

func DeleteMembership(db *gorm.DB, userID uuid.UUID, bucketID uuid.UUID) error {
	return db.Where("user_id = ? AND bucket_id = ?", userID, bucketID).
		Delete(&models.Membership{}).Error
}

func HasBucketAccess(
	db *gorm.DB,
	userID uuid.UUID,
	bucketID uuid.UUID,
	requiredGroup models.Group,
) (bool, error) {
	membership, err := GetUserMembership(db, userID, bucketID)
	if err != nil {
		return false, err
	}
	if membership == nil {
		return false, nil
	}
	return HasGroup(membership.Group, requiredGroup), nil
}
