//go:build integration

package integration

import (
	"os"
	"testing"

	tclog "github.com/testcontainers/testcontainers-go/log"
)

func TestMain(m *testing.M) {
	if !integrationVerbose() {
		tclog.SetDefault(silentTCLogger{})
		_ = os.Setenv("TESTCONTAINERS_RYUK_VERBOSE", "false")
	}
	os.Exit(m.Run())
}

type silentTCLogger struct{}

func (silentTCLogger) Printf(string, ...any) {}
