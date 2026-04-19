//go:build integration

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/safebucket/safebucket/internal/middlewares"
)

const testMaxUploadSize = 32 << 20

func TestMain(m *testing.M) {
	backend := os.Getenv("INTEGRATION_DB")
	if backend == "" {
		backend = "postgres"
	}

	provider, err := newProvider(backend)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration: %v\n", err)
		os.Exit(1)
	}
	testProvider = provider

	middlewares.InitValidator(testMaxUploadSize)

	code := m.Run()

	provider.Teardown()
	os.Exit(code)
}

func newProvider(backend string) (DBProvider, error) {
	switch backend {
	case "sqlite":
		return &SQLiteProvider{}, nil
	case "postgres":
		return &PostgresProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown INTEGRATION_DB backend %q (want \"postgres\" or \"sqlite\")", backend)
	}
}
