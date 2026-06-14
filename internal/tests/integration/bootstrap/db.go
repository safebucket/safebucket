//go:build integration

package bootstrap

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/safebucket/safebucket/internal/database"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DBProvider interface {
	Connect(t *testing.T) *gorm.DB
	Setup(t *testing.T) *gorm.DB
	Dialect() string
	Teardown()
}

type SQLiteProvider struct{}

func (p *SQLiteProvider) Connect(t *testing.T) *gorm.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open("file:"+path+"?_txlock=immediate"), &gorm.Config{Logger: gormTestLogger()})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.ExecContext(context.Background(), "PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	t.Cleanup(func() { _ = sqlDB.Close() })

	return db
}

func (p *SQLiteProvider) Setup(t *testing.T) *gorm.DB {
	t.Helper()

	db := p.Connect(t)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	database.RunMigrations(sqlDB, database.DialectSQLite)
	database.RegisterCallbacks(db)

	return db
}

func (p *SQLiteProvider) Dialect() string {
	return database.DialectSQLite
}

func (p *SQLiteProvider) Teardown() {}
