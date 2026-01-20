package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"api/internal/tests"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mfaTestJWTSecret = "test-secret-key-for-mfa-testing"

// generateRestrictedToken creates a restricted access token for testing.
func generateRestrictedToken(secret string, user *models.User, audience string, mfaVerified bool) (string, error) {
	return helpers.NewRestrictedAccessToken(secret, user, audience, mfaVerified, nil)
}

// generateFullAccessToken creates a full access token for testing.
func generateFullAccessToken(secret string, user *models.User) (string, error) {
	return helpers.NewAccessToken(secret, user, string(models.LocalProviderType))
}

// TestMFAValidate_MFAEnforcement tests MFA enforcement for full access tokens.
// Note: Audience validation is now handled by AudienceValidate middleware, not MFAValidate.
func TestMFAValidate_MFAEnforcement(t *testing.T) {
	t.Run("should require MFA setup for local user without MFA when mfaRequired is true", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			// No MFA devices = claims.MFA will be false
		}

		token, err := generateFullAccessToken(mfaTestJWTSecret, testUser)
		require.NoError(t, err)

		// Access a protected endpoint that requires MFA
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		// Parse token and set up context
		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow user with MFA enabled when mfaRequired is true", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			MFADevices: []models.MFADevice{
				{ID: uuid.New(), IsVerified: true}, // User has verified MFA device
			},
		}

		token, err := generateFullAccessToken(mfaTestJWTSecret, testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should not require MFA when mfaRequired is false", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			// No MFA devices
		}

		token, err := generateFullAccessToken(mfaTestJWTSecret, testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when MFA not required")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

// TestMFAValidate_RestrictedTokensSkipMFAEnforcement tests that restricted tokens
// (auth:mfa:* audiences) skip MFA enforcement since they're in the MFA flow.
func TestMFAValidate_RestrictedTokensSkipMFAEnforcement(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
		// No MFA devices - would normally trigger FORBIDDEN for full access tokens
	}

	t.Run("should skip MFA enforcement for restricted tokens (handled by AudienceValidate)", func(t *testing.T) {
		// Restricted token with mfa:login audience
		token, err := generateRestrictedToken(mfaTestJWTSecret, testUser, configuration.AudienceMFALogin, false)
		require.NoError(t, err)

		// Access MFA device endpoint (allowed by AudienceValidate for this token)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/mfa/devices", nil)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		// MFAValidate should pass because the token is NOT app:* audience
		// (MFA enforcement only applies to app:* tokens)
		assert.True(t, nextCalled, "Next handler should be called for restricted tokens")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_NoClaims(t *testing.T) {
	t.Run("should return FORBIDDEN when no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		// No claims set in context (simulates middleware chain error)
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})
}

func TestMFAValidate_AuthExcluded(t *testing.T) {
	t.Run("should skip validation when auth is excluded", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		recorder := httptest.NewRecorder()

		// Set auth excluded flag (as Authenticate middleware would)
		ctx := context.WithValue(req.Context(), AuthExcludedKey{}, true)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for excluded paths")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

// TestMFAValidate_OAuthUsersSkipMFA tests that OAuth users don't require MFA.
func TestMFAValidate_OAuthUsersSkipMFA(t *testing.T) {
	t.Run("should not require MFA for OAuth users even when mfaRequired is true", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "oauth@example.com",
			Role:         models.RoleUser,
			ProviderType: models.OIDCProviderType,
			// No MFA devices, but OAuth provider
		}

		// Generate token with OAuth provider
		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, "google")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		// OAuth users should bypass MFA enforcement
		assert.True(t, nextCalled, "Next handler should be called for OAuth users")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestIsMFABypassPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{
			name:     "Regular bucket path is not bypass",
			path:     "/api/v1/buckets",
			method:   "GET",
			expected: false,
		},
		{
			name:     "User profile is not bypass",
			path:     "/api/v1/users/123",
			method:   "GET",
			expected: false,
		},
		{
			name:     "MFA devices path is not bypass",
			path:     "/api/v1/users/123/mfa/devices",
			method:   "GET",
			expected: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := isMFABypassPath(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMFAValidate_TokenUserStateMismatch(t *testing.T) {
	t.Run("should enforce MFA when user has devices but token claims MFA false", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(mfaTestJWTSecret, testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)
		assert.False(t, claims.MFA, "Token should have MFA=false since user had no devices at token creation")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow when claims MFA is true even if devices were removed", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			MFADevices: []models.MFADevice{
				{ID: uuid.New(), IsVerified: true},
			},
		}

		token, err := generateFullAccessToken(mfaTestJWTSecret, testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)
		assert.True(t, claims.MFA, "Token should have MFA=true since user had verified device at token creation")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Should allow access with MFA=true token regardless of current device state")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_ProviderTypeMismatch(t *testing.T) {
	t.Run("should handle OAuth provider with local provider type claim", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.OIDCProviderType,
		}

		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, "google")
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "OAuth users should bypass MFA enforcement")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_CrossFlowTokenAccess(t *testing.T) {
	t.Run("password reset token accessing non-MFA-enforced route should pass MFAValidate", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateRestrictedToken(mfaTestJWTSecret, testUser, configuration.AudienceMFAReset, false)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "MFAValidate should pass restricted tokens (audience validation is separate)")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
