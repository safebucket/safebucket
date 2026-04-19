//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/safebucket/safebucket/internal/database"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	defaultPostgresImage = "postgres:17-alpine"
	postgresDBName       = "safebucket_test"
	postgresUser         = "safebucket"
	postgresPassword     = "safebucket"
)

type PostgresProvider struct{}

func (p *PostgresProvider) Setup(t *testing.T) *gorm.DB {
	t.Helper()

	ctx := context.Background()

	image := os.Getenv("POSTGRES_IMAGE")
	if image == "" {
		image = defaultPostgresImage
	}

	container, err := tcpostgres.Run(ctx, image,
		tcpostgres.WithDatabase(postgresDBName),
		tcpostgres.WithUsername(postgresUser),
		tcpostgres.WithPassword(postgresPassword),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start postgres container")

	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "postgres connection string")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	t.Cleanup(func() { _ = sqlDB.Close() })

	database.RunMigrations(sqlDB, database.DialectPostgres)
	database.RegisterCallbacks(db)

	return db
}

func (p *PostgresProvider) Dialect() string {
	return database.DialectPostgres
}

func (p *PostgresProvider) Teardown() {}
