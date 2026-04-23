//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/safebucket/safebucket/internal/database"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DBProvider interface {
	Setup(t *testing.T) *gorm.DB
	Dialect() string
	Teardown()
}

type SQLiteProvider struct{}

func (p *SQLiteProvider) Setup(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	// :memory: gives each connection its own empty database, so pin the pool to
	// a single connection (matches internal/database/sqlite.go).
	sqlDB.SetMaxOpenConns(1)

	_, err = sqlDB.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	database.RunMigrations(sqlDB, database.DialectSQLite)
	database.RegisterCallbacks(db)

	t.Cleanup(func() { _ = sqlDB.Close() })

	return db
}

func (p *SQLiteProvider) Dialect() string {
	return database.DialectSQLite
}

func (p *SQLiteProvider) Teardown() {}
