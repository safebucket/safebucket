//go:build integration

package bucket_test

import (
	"os"
	"testing"

	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"
)

func TestMain(m *testing.M) {
	os.Exit(bootstrap.RunTestMain(m))
}
