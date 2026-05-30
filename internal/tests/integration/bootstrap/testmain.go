//go:build integration

package bootstrap

import (
	"os"
	"testing"

	tclog "github.com/testcontainers/testcontainers-go/log"
)

func RunTestMain(m *testing.M) int {
	if !integrationVerbose() {
		tclog.SetDefault(silentTCLogger{})
		_ = os.Setenv("TESTCONTAINERS_RYUK_VERBOSE", "false")
	}
	return m.Run()
}

type silentTCLogger struct{}

func (silentTCLogger) Printf(string, ...any) {}
