//go:build integration

package user_test

import (
	"os"
	"testing"

	"github.com/safebucket/safebucket/internal/tests/integration/harness"
)

func TestMain(m *testing.M) {
	os.Exit(harness.RunTestMain(m))
}
