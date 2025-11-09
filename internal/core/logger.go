package core

import (
	"errors"
	"runtime"
	"strings"
	"syscall"

	"go.uber.org/zap"
)

func NewLogger(level string) {
	config := zap.NewProductionConfig()

	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	case "panic":
		config.Level = zap.NewAtomicLevelAt(zap.PanicLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	zap.ReplaceGlobals(logger)

	defer func(logger *zap.Logger) {
		err = logger.Sync()
		if err != nil && !isIgnorableLogSyncError(err) {
			panic(err)
		}
	}(logger)
}

// isIgnorableLogSyncError returns true for errors that can be safely ignored during logger sync.
func isIgnorableLogSyncError(err error) bool {
	// Standard UNIX not-a-terminal error
	if errors.Is(err, syscall.ENOTTY) {
		return true
	}

	// Invalid argument errors (common when stderr is redirected or closed)
	if errors.Is(err, syscall.EINVAL) {
		return true
	}

	// Check error string for common sync errors across all platforms
	errStr := err.Error()
	if strings.Contains(errStr, "sync /dev/stderr") ||
		strings.Contains(errStr, "sync /dev/stdout") ||
		strings.Contains(errStr, "inappropriate ioctl for device") ||
		strings.Contains(errStr, "invalid argument") {
		return true
	}

	// Windows-specific sync errors
	if runtime.GOOS == "windows" {
		if strings.Contains(errStr, "The handle is invalid") {
			return true
		}
	}

	return false
}
