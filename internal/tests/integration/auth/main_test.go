//go:build integration

package auth_test

import (
	"os"
	"testing"

	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"
)

func TestMain(m *testing.M) {
	os.Exit(bootstrap.RunTestMain(m))
}
