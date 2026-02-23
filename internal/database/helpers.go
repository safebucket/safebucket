package database

import (
	"api/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FormatDateStr returns the correct SQL expression to extract a formatted date (YYYY-MM-DD)
// depending on the underlying database dialect.
// IMPORTANT: column must be a trusted, hardcoded column identifier — it is interpolated
// directly into the SQL string without escaping.
func FormatDateStr(db *gorm.DB, column string) string {
	if db.Dialector.Name() == DialectSQLite {
		return "strftime('%Y-%m-%d', " + column + ")"
	}
	// Fallback to PostgreSQL syntax
	return "TO_CHAR(" + column + ", 'YYYY-MM-DD')"
}

// UpsertAdminUser creates or updates the admin user based on the database dialect
// to handle constraint conflicts correctly.
func UpsertAdminUser(db *gorm.DB, adminUser *models.User) {
	if db.Dialector.Name() == DialectSQLite {
		if err := db.Transaction(func(tx *gorm.DB) error {
			var existing models.User
			result := tx.Where("email = ? AND provider_key = ? AND deleted_at IS NULL",
				adminUser.Email, adminUser.ProviderKey).First(&existing)
			if result.Error == nil {
				return tx.Model(&existing).Update("hashed_password", adminUser.HashedPassword).Error
			}
			return tx.Create(adminUser).Error
		}); err != nil {
			zap.L().Fatal("Failed to upsert admin user", zap.Error(err))
		}
	} else {
		if err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "email"}, {Name: "provider_key"}},
			TargetWhere: clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: "deleted_at", Value: nil},
			}},
			DoUpdates: clause.AssignmentColumns([]string{"hashed_password"}),
		}).Create(adminUser).Error; err != nil {
			zap.L().Fatal("Failed to upsert admin user", zap.Error(err))
		}
	}
}
