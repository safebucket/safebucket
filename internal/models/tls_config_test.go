package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSEnabled(t *testing.T) {
	t.Run("returns false when both fields are empty", func(t *testing.T) {
		cfg := AppConfiguration{}
		assert.False(t, cfg.TLSEnabled())
	})

	t.Run("returns true when both fields are set", func(t *testing.T) {
		cfg := AppConfiguration{
			TLSCertFile: "/path/to/cert.pem",
			TLSKeyFile:  "/path/to/key.pem",
		}
		assert.True(t, cfg.TLSEnabled())
	})
}
