package helpers

import (
	"strings"
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenPrefixForProvider(t *testing.T) {
	assert.Equal(t, TokenPrefixServiceAccount, TokenPrefixForProvider(models.ServiceAccountProviderType))
	assert.Equal(t, TokenPrefixPersonal, TokenPrefixForProvider(models.LocalProviderType))
	assert.Equal(t, TokenPrefixPersonal, TokenPrefixForProvider(models.OIDCProviderType))
}

func TestGenerateAPIToken(t *testing.T) {
	t.Run("personal token round-trips through parse", func(t *testing.T) {
		token, hash, err := GenerateAPIToken(models.LocalProviderType)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(token, TokenPrefixPersonal))
		assert.NotEmpty(t, hash)

		parsedHash, parseErr := ParseAPIToken(token)
		require.NoError(t, parseErr)
		assert.Equal(t, hash, parsedHash)
	})

	t.Run("service-account token gets the sat prefix", func(t *testing.T) {
		token, hash, err := GenerateAPIToken(models.ServiceAccountProviderType)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(token, TokenPrefixServiceAccount))

		parsedHash, parseErr := ParseAPIToken(token)
		require.NoError(t, parseErr)
		assert.Equal(t, hash, parsedHash)
	})

	t.Run("two tokens never collide", func(t *testing.T) {
		t1, h1, err1 := GenerateAPIToken(models.LocalProviderType)
		t2, h2, err2 := GenerateAPIToken(models.LocalProviderType)
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, t1, t2)
		assert.NotEqual(t, h1, h2)
	})

	t.Run("hash is not reversible to the token", func(t *testing.T) {
		token, hash, err := GenerateAPIToken(models.LocalProviderType)
		require.NoError(t, err)
		assert.NotContains(t, hash, strings.TrimPrefix(token, TokenPrefixPersonal))
	})
}

func TestHasAPITokenPrefix(t *testing.T) {
	assert.True(t, HasAPITokenPrefix("sb_pat_abc"))
	assert.True(t, HasAPITokenPrefix("sb_sat_abc"))
	assert.False(t, HasAPITokenPrefix("Bearer something"))
	assert.False(t, HasAPITokenPrefix("sb_xxx_abc"))
	assert.False(t, HasAPITokenPrefix(""))
}

func TestParseAPIToken(t *testing.T) {
	t.Run("rejects unknown prefix", func(t *testing.T) {
		_, err := ParseAPIToken("sb_xxx_deadbeef")
		require.Error(t, err)
	})

	t.Run("rejects wrong length", func(t *testing.T) {
		_, err := ParseAPIToken(TokenPrefixPersonal + "tooshort")
		require.Error(t, err)
	})

	t.Run("rejects a corrupted checksum (typo)", func(t *testing.T) {
		token, _, err := GenerateAPIToken(models.LocalProviderType)
		require.NoError(t, err)

		runes := []rune(token)
		last := runes[len(runes)-1]
		if last == 'a' {
			runes[len(runes)-1] = 'b'
		} else {
			runes[len(runes)-1] = 'a'
		}

		_, parseErr := ParseAPIToken(string(runes))
		require.Error(t, parseErr)
	})

	t.Run("valid token produces the same hash as generation", func(t *testing.T) {
		token, hash, err := GenerateAPIToken(models.ServiceAccountProviderType)
		require.NoError(t, err)

		parsed, parseErr := ParseAPIToken(token)
		require.NoError(t, parseErr)
		assert.Equal(t, hash, parsed)
	})
}
