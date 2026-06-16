package database

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FormatHourStr returns the SQL expression that formats a timestamp column as an
// RFC3339 UTC hour bucket (e.g. 2026-06-16T14:00:00Z) for the underlying dialect.
func FormatHourStr(db *gorm.DB, column string) string {
	if db.Dialector.Name() == DialectSQLite {
		return fmt.Sprintf("strftime('%%Y-%%m-%%dT%%H:00:00Z', %s)", column)
	}
	return fmt.Sprintf("TO_CHAR(%s, 'YYYY-MM-DD\"T\"HH24\":00:00Z\"')", column)
}

func UpsertAdminUser(db *gorm.DB, adminUser *models.User) {
	if err := db.Transaction(func(tx *gorm.DB) error {
		var existing models.User
		result := tx.Where("email = ? AND provider_key = ? AND deleted_at IS NULL",
			adminUser.Email, adminUser.ProviderKey).Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			return tx.Model(&existing).Update("hashed_password", adminUser.HashedPassword).Error
		}
		return tx.Create(adminUser).Error
	}); err != nil {
		zap.L().Fatal("Failed to upsert admin user", zap.Error(err))
	}
}
