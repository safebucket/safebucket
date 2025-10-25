package sql

import (
	c "api/internal/configuration"
	"api/internal/errors"
	"api/internal/models"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetById[T any](db *gorm.DB, id uuid.UUID) (T, error) {
	var obj T
	result := db.Where("id = ?", id).First(&obj)
	if result.RowsAffected == 0 {
		return *new(T), errors.NewAPIError(404, "NOT_FOUND")
	}

	return obj, nil
}

// WithCasbinTx executes a function within a GORM transaction with a transaction-aware Casbin enforcer.
//
// This function ensures that both database operations and Casbin policy changes are atomic and will be
// rolled back together if an error occurs. It creates a new Casbin enforcer instance that uses the
// transactional GORM adapter, ensuring policy changes are written to the same transaction.
//
// Key behaviors:
//   - Creates a GORM transaction wrapper around the provided callback function
//   - Instantiates a new GORM adapter using the transaction context (tx) instead of the base DB
//   - Creates a new Casbin enforcer instance with the same model but transactional adapter
//   - If the callback returns an error, both DB changes AND Casbin policy changes are rolled back
//   - If the callback returns nil, both DB changes AND Casbin policy changes are committed atomically
func WithCasbinTx(db *gorm.DB, enforcer *casbin.Enforcer, fn func(*gorm.DB, *casbin.Enforcer) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		txAdapter, err := gormadapter.NewAdapterByDBWithCustomTable(tx, &models.Policy{}, c.PolicyTableName)
		if err != nil {
			return err
		}

		txEnforcer, err := casbin.NewEnforcer(enforcer.GetModel(), txAdapter)
		if err != nil {
			return err
		}

		return fn(tx, txEnforcer)
	})
}
