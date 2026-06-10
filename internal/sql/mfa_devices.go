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
