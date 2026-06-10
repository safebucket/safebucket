package configuration

import (
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvidersLocal(t *testing.T) {
	t.Run("finds the local provider and its key regardless of naming", func(t *testing.T) {
		providers := Providers{
			"passwords": {Type: models.LocalProviderType, MFARequired: true},
			"okta":      {Type: models.OIDCProviderType},
		}

		key, provider, ok := providers.Local()

		require.True(t, ok)
		assert.Equal(t, "passwords", key)
		assert.Equal(t, models.LocalProviderType, provider.Type)
		assert.True(t, provider.MFARequired)
	})

	t.Run("returns false when no local provider is configured", func(t *testing.T) {
		providers := Providers{"okta": {Type: models.OIDCProviderType}}

		_, _, ok := providers.Local()

		assert.False(t, ok)
	})
}

func TestValidateProviderKeys(t *testing.T) {
	t.Run("rejects a non-local provider under the reserved local key", func(t *testing.T) {
		providers := map[string]models.ProviderConfiguration{
			"local": {Type: models.OIDCProviderType},
		}

		err := validateProviderKeys(providers)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reserved")
	})

	t.Run("accepts the local provider under the local key", func(t *testing.T) {
		providers := map[string]models.ProviderConfiguration{
			"local": {Type: models.LocalProviderType},
		}

		assert.NoError(t, validateProviderKeys(providers))
	})

	t.Run("accepts the local provider under a custom key", func(t *testing.T) {
		providers := map[string]models.ProviderConfiguration{
			"passwords": {Type: models.LocalProviderType},
			"okta":      {Type: models.OIDCProviderType},
		}

		assert.NoError(t, validateProviderKeys(providers))
	})
}
