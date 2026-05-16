package mfa

import (
	"testing"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const mfaTestJWTSecret = "mfa-test-secret-key-32-bytes-xxx"

func newMFATestUser() *models.User {
	return &models.User{
		ID:           uuid.New(),
		Email:        "mfatest@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}
}

func newMFATestConfig() models.AuthConfig {
	return models.AuthConfig{
		TokenSecret: mfaTestJWTSecret,
	}
}

func TestHandleMFARequired(t *testing.T) {
	logger := zap.NewNop()

	t.Run("returns a non-empty restricted access token", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		token, err := HandleMFARequired(logger, cfg, user)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("token has MFA login audience", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		token, err := HandleMFARequired(logger, cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+token, true)
		require.NoError(t, parseErr)
		assert.Equal(t, configuration.AudienceMFALogin, claims.AudienceString())
	})

	t.Run("token embeds correct user identity", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		token, err := HandleMFARequired(logger, cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+token, true)
		require.NoError(t, parseErr)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
	})

	t.Run("token MFA flag is false before verification", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		token, err := HandleMFARequired(logger, cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+token, true)
		require.NoError(t, parseErr)
		assert.False(t, claims.MFA)
	})

	t.Run("token is parseable by the configured secret", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		token, err := HandleMFARequired(logger, cfg, user)
		require.NoError(t, err)

		_, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+token, true)
		require.NoError(t, parseErr)
	})
}

func TestGenerateTokens(t *testing.T) {
	t.Run("returns non-empty SID and both tokens", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		sid, tokens, err := GenerateTokens(cfg, user)

		require.NoError(t, err)
		assert.NotEmpty(t, sid)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("access token has full access audience", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		_, tokens, err := GenerateTokens(cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+tokens.AccessToken, true)
		require.NoError(t, parseErr)
		assert.Equal(t, configuration.AudienceAccessToken, claims.AudienceString())
	})

	t.Run("refresh token has refresh audience", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		_, tokens, err := GenerateTokens(cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseRefreshToken(cfg.TokenSecret, tokens.RefreshToken)
		require.NoError(t, parseErr)
		assert.Equal(t, configuration.AudienceRefreshToken, claims.AudienceString())
	})

	t.Run("SID is embedded in both tokens", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		sid, tokens, err := GenerateTokens(cfg, user)
		require.NoError(t, err)

		accessClaims, err := helpers.ParseToken(cfg.TokenSecret, "Bearer "+tokens.AccessToken, true)
		require.NoError(t, err)
		assert.Equal(t, sid, accessClaims.SID)

		refreshClaims, err := helpers.ParseRefreshToken(cfg.TokenSecret, tokens.RefreshToken)
		require.NoError(t, err)
		assert.Equal(t, sid, refreshClaims.SID)
	})

	t.Run("both tokens carry the user identity", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		_, tokens, err := GenerateTokens(cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, "Bearer "+tokens.AccessToken, true)
		require.NoError(t, parseErr)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("each call produces a different SID and tokens", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		sid1, tokens1, err1 := GenerateTokens(cfg, user)
		sid2, tokens2, err2 := GenerateTokens(cfg, user)
		require.NoError(t, err1)
		require.NoError(t, err2)

		assert.NotEqual(t, sid1, sid2)
		assert.NotEqual(t, tokens1.AccessToken, tokens2.AccessToken)
		assert.NotEqual(t, tokens1.RefreshToken, tokens2.RefreshToken)
	})

	t.Run("access token is signed with HS256", func(t *testing.T) {
		user := newMFATestUser()
		cfg := newMFATestConfig()

		_, tokens, err := GenerateTokens(cfg, user)
		require.NoError(t, err)

		claims, parseErr := helpers.ParseToken(cfg.TokenSecret, tokens.AccessToken, false)
		require.NoError(t, parseErr)
		assert.Equal(t, "app:*", claims.Audience[0])
	})
}
