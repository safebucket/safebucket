//go:build integration

package bootstrap

import (
	"log"
	"os"

	"gorm.io/gorm/logger"
)

func integrationVerbose() bool {
	return os.Getenv("INTEGRATION_VERBOSE") != ""
}

func gormTestLogger() logger.Interface {
	if integrationVerbose() {
		return logger.New(log.New(os.Stderr, "\r\n", log.LstdFlags), logger.Config{
			LogLevel: logger.Warn,
		})
	}
	return logger.Default.LogMode(logger.Silent)
}
