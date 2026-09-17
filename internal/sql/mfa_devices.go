package sql

import (
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CountVerifiedMFADevices(db *gorm.DB, userID uuid.UUID) (int64, error) {
	var count int64
	result := db.Model(&models.MFADevice{}).
		Where("user_id = ? AND is_verified = ?", userID, true).
		Count(&count)
	return count, result.Error
}

func VerifiedMFAUserIDs(db *gorm.DB) (map[uuid.UUID]bool, error) {
	var ids []uuid.UUID
	result := db.Model(&models.MFADevice{}).
		Where("is_verified = ?", true).
		Distinct().
		Pluck("user_id", &ids)
	if result.Error != nil {
		return nil, result.Error
	}

	enabled := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		enabled[id] = true
	}
	return enabled, nil
}
