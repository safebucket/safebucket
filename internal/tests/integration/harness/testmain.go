//go:build integration

package harness

import (
	"os"
	"testing"

	tclog "github.com/testcontainers/testcontainers-go/log"
)

// RunTestMain is the entrypoint every feature subpackage's TestMain delegates to.
// It silences testcontainers' logger unless INTEGRATION_VERBOSE is set, then runs
// the package's tests. Returning the exit code lets callers `os.Exit` themselves.
func RunTestMain(m *testing.M) int {
	if !integrationVerbose() {
		tclog.SetDefault(silentTCLogger{})
		_ = os.Setenv("TESTCONTAINERS_RYUK_VERBOSE", "false")
	}
	return m.Run()
}

type silentTCLogger struct{}

func (silentTCLogger) Printf(string, ...any) {}
