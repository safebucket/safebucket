package configuration

import (
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestConfig(t *testing.T, yaml string) models.Configuration {
	t.Helper()
	cfg, err := Load(LoadOptions{ConfigBytes: []byte(yaml), SkipEnv: true})
	require.NoError(t, err)
	return cfg
}

func TestLoadSplitsCommaSeparatedListsFromEnv(t *testing.T) {
	t.Setenv("APP__ALLOWED_ORIGINS", "http://a:1, http://b:2")
	t.Setenv("APP__TRUSTED_PROXIES", "10.0.0.0/8,192.168.0.0/16")
	t.Setenv("CACHE__REDIS__HOSTS", "redis-1:6379 redis-2:6379")

	cfg, err := Load(LoadOptions{ConfigBytes: []byte("app:\n  port: 8080\n")})
	require.NoError(t, err)

	assert.Equal(t, []string{"http://a:1", "http://b:2"}, cfg.App.AllowedOrigins)
	assert.Equal(t, []string{"10.0.0.0/8", "192.168.0.0/16"}, cfg.App.TrustedProxies)
	assert.Equal(t, []string{"redis-1:6379", "redis-2:6379"}, cfg.Cache.Redis.Hosts)
}

func TestLoadKeepsYAMLLists(t *testing.T) {
	cfg := loadTestConfig(t, "app:\n  allowed_origins:\n    - http://a:1\n    - http://b:2\n")

	assert.Equal(t, []string{"http://a:1", "http://b:2"}, cfg.App.AllowedOrigins)
}
