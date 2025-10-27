package rbac

import (
	"api/internal/models"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetUserMembership returns the user's membership for a specific bucket
// Returns nil if no membership exists.
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

// GetBucketMembers returns all memberships for a specific bucket.
func GetBucketMembers(db *gorm.DB, bucketID uuid.UUID) ([]models.Membership, error) {
	var memberships []models.Membership
	err := db.Where("bucket_id = ?", bucketID).Preload("User").Find(&memberships).Error
	return memberships, err
}

// GetUserBuckets returns all bucket memberships for a specific user.
func GetUserBuckets(db *gorm.DB, userID uuid.UUID) ([]models.Membership, error) {
	var memberships []models.Membership
	err := db.Where("user_id = ?", userID).Preload("Bucket").Find(&memberships).Error
	return memberships, err
}

// CreateMembership creates a new membership record.
func CreateMembership(db *gorm.DB, userID uuid.UUID, bucketID uuid.UUID, group models.Group) error {
	membership := models.Membership{
		UserID:   userID,
		BucketID: bucketID,
		Group:    group,
	}
	return db.Create(&membership).Error
}

// UpdateMembership updates an existing membership's group.
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

// DeleteMembership removes a membership record.
func DeleteMembership(db *gorm.DB, userID uuid.UUID, bucketID uuid.UUID) error {
	return db.Where("user_id = ? AND bucket_id = ?", userID, bucketID).
		Delete(&models.Membership{}).Error
}

// HasBucketAccess checks if a user has at least the required group access to a bucket.
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
