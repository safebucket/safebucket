//go:build integration

package bootstrap

import (
	"database/sql"
	"path/filepath"
	"runtime"

	"github.com/safebucket/safebucket/internal/database"

	"github.com/pressly/goose/v3"
)

func migrationsDir(dialect string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "database", "migrations", dialect)
}

func MigrationVersions(dialect string) ([]int64, error) {
	migrations, err := goose.CollectMigrations(migrationsDir(dialect), 0, goose.MaxVersion)
	if err != nil {
		return nil, err
	}

	versions := make([]int64, len(migrations))
	for i, m := range migrations {
		versions[i] = m.Version
	}
	return versions, nil
}

func MigrateUpTo(db *sql.DB, dialect string, version int64) error {
	gooseDialect := dialect
	if dialect == database.DialectSQLite {
		gooseDialect = "sqlite3"
	}
	if err := goose.SetDialect(gooseDialect); err != nil {
		return err
	}
	return goose.UpTo(db, migrationsDir(dialect), version)
}
