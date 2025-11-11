package core

import (
	"errors"
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

	errStr := err.Error()
	// Common Windows stderr sync errors
	if strings.Contains(errStr, "The handle is invalid") ||
		strings.Contains(errStr, "sync /dev/stderr") ||
		strings.Contains(errStr, "inappropriate ioctl for device") {
		return true
	}

	return false
}
