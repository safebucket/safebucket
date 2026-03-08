package database

import (
	"fmt"

	"github.com/safebucket/safebucket/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FormatDateStr returns the correct SQL expression to extract a formatted date (YYYY-MM-DD)
// depending on the underlying database dialect.
// IMPORTANT: column must be a trusted, hardcoded column identifier — it is interpolated
// directly into the SQL string without escaping.
func FormatDateStr(db *gorm.DB, column string) string {
	if db.Dialector.Name() == DialectSQLite {
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s)", column)
	}
	// Fallback to PostgreSQL syntax
	return fmt.Sprintf("TO_CHAR(%s, 'YYYY-MM-DD')", column)
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
