package middlewares

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"api/internal/configuration"
	"api/internal/helpers"
	"api/internal/models"
	"api/internal/tests"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const mfaTestJWTSecret = "test-secret-key-for-mfa-testing"

// generateRestrictedToken creates a restricted access token for testing.
func generateRestrictedToken(secret string, user *models.User, audience string, mfaVerified bool) (string, error) {
	return helpers.NewRestrictedAccessToken(secret, user, audience, mfaVerified, nil)
}

// generateFullAccessToken creates a full access token for testing.
func generateFullAccessToken(user *models.User) (string, error) {
	return helpers.NewAccessToken(mfaTestJWTSecret, user, string(models.LocalProviderType))
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

		token, err := generateFullAccessToken(testUser)
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

		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		token, err := generateFullAccessToken(testUser)
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		token, err := generateFullAccessToken(testUser)
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
		handler := MFAValidate(nil, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Aud)
		assert.False(t, claims.MFA, "Token should have MFA=false since user had no devices at token creation")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		token, err := generateFullAccessToken(testUser)
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		handler := MFAValidate(nil, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "MFAValidate should pass restricted tokens (audience validation is separate)")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestIsMFABypassPath_WithConfiguredRules(t *testing.T) {
	// Save original rules and restore after test
	originalRules := configuration.MFABypassRules
	defer func() { configuration.SetMFABypassRulesForTesting(originalRules) }()

	t.Run("exact path match with specific method", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{ExactPath: "/api/v1/health", Method: "GET"},
		})
		assert.True(t, isMFABypassPath("/api/v1/health", "GET"), "should match exact path with matching method")
		assert.False(t, isMFABypassPath("/api/v1/health", "POST"), "should not match exact path with different method")
		assert.False(t, isMFABypassPath("/api/v1/other", "GET"), "should not match different path")
	})

	t.Run("pattern match with wildcard method", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{Pattern: regexp.MustCompile(`^/api/v1/public/.*$`), Method: "*"},
		})
		assert.True(t, isMFABypassPath("/api/v1/public/resource", "GET"), "should match pattern with GET")
		assert.True(t, isMFABypassPath("/api/v1/public/resource", "POST"), "should match pattern with POST")
		assert.True(t, isMFABypassPath("/api/v1/public/nested/resource", "DELETE"),
			"should match nested path with DELETE")
		assert.False(t, isMFABypassPath("/api/v1/private/resource", "GET"), "should not match non-conforming path")
	})

	t.Run("wildcard method on exact path", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{ExactPath: "/api/v1/open", Method: "*"},
		})
		assert.True(t, isMFABypassPath("/api/v1/open", "GET"), "should match with GET")
		assert.True(t, isMFABypassPath("/api/v1/open", "POST"), "should match with POST")
		assert.True(t, isMFABypassPath("/api/v1/open", "DELETE"), "should match with DELETE")
		assert.True(t, isMFABypassPath("/api/v1/open", "PATCH"), "should match with PATCH")
	})

	t.Run("pattern match with specific method", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{Pattern: regexp.MustCompile(`^/api/v1/readonly/.*$`), Method: "GET"},
		})
		assert.True(t, isMFABypassPath("/api/v1/readonly/data", "GET"), "should match pattern with GET")
		assert.False(t, isMFABypassPath("/api/v1/readonly/data", "POST"), "should not match pattern with POST")
		assert.False(t, isMFABypassPath("/api/v1/readonly/data", "DELETE"), "should not match pattern with DELETE")
	})

	t.Run("multiple rules", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{ExactPath: "/api/v1/health", Method: "GET"},
			{Pattern: regexp.MustCompile(`^/api/v1/public/.*$`), Method: "*"},
			{ExactPath: "/api/v1/status", Method: "GET"},
		})
		assert.True(t, isMFABypassPath("/api/v1/health", "GET"), "should match first rule")
		assert.True(t, isMFABypassPath("/api/v1/public/anything", "POST"), "should match second rule")
		assert.True(t, isMFABypassPath("/api/v1/status", "GET"), "should match third rule")
		assert.False(t, isMFABypassPath("/api/v1/protected", "GET"), "should not match any rule")
	})

	t.Run("UUID pattern matching", func(t *testing.T) {
		configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
			{
				Pattern: regexp.MustCompile(`^/api/v1/resources/` + configuration.UUIDv4Pattern + `/info$`),
				Method:  "GET",
			},
		})
		assert.True(t, isMFABypassPath("/api/v1/resources/550e8400-e29b-41d4-a716-446655440000/info", "GET"),
			"should match valid UUID path")
		assert.False(t, isMFABypassPath("/api/v1/resources/invalid-uuid/info", "GET"),
			"should not match invalid UUID path")
		assert.False(t, isMFABypassPath("/api/v1/resources/550e8400-e29b-41d4-a716-446655440000/other", "GET"),
			"should not match different suffix")
	})
}

func TestMFAValidate_BypassPath(t *testing.T) {
	// Save original rules and restore after test
	originalRules := configuration.MFABypassRules
	defer func() { configuration.SetMFABypassRulesForTesting(originalRules) }()

	configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
		{ExactPath: "/api/v1/health", Method: "GET"},
	})

	t.Run("should allow user without MFA on bypass path", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			// No MFA devices = claims.MFA will be false
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		// Access bypass path
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.False(t, claims.MFA, "claims.MFA should be false for user without MFA devices")

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for bypass path")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should block user without MFA on non-bypass path", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			// No MFA devices = claims.MFA will be false
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		// Access non-bypass path
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should block user without MFA on bypass path with wrong method", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		// Access bypass path with wrong method (POST instead of GET)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})
}

// TestMFAValidate_EnrollmentCheck tests the database enrollment check functionality.
// This verifies that users with stale tokens (MFA=false but enrolled in DB) are blocked.
func TestMFAValidate_EnrollmentCheck(t *testing.T) {
	t.Run("should block local user with MFA=false token when user has enrolled MFA", func(t *testing.T) {
		// Setup mock DB
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			// No MFA devices at token creation time
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.False(t, claims.MFA, "Token should have MFA=false")

		// Simulate user enrolling MFA after token was issued
		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		// mfaRequired=false but DB has enrolled device
		handler := MFAValidate(gormDB, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should allow local user with MFA=false token when user has no MFA enrolled", func(t *testing.T) {
		// Setup mock DB
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		// User has no MFA devices in DB
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(gormDB, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when user has no MFA enrolled")
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should not check DB for OAuth users", func(t *testing.T) {
		// Setup mock DB - no expectations should be set
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "oauth@example.com",
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
		handler := MFAValidate(gormDB, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for OAuth users without DB check")
		assert.Equal(t, http.StatusOK, recorder.Code)
		// No expectations set - mock will fail if any DB call was made
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should not check DB when token has MFA=true", func(t *testing.T) {
		// Setup mock DB - no expectations should be set
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
			MFADevices: []models.MFADevice{
				{ID: uuid.New(), IsVerified: true},
			},
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.True(t, claims.MFA, "Token should have MFA=true")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(gormDB, false)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when token has MFA=true without DB check")
		assert.Equal(t, http.StatusOK, recorder.Code)
		// No expectations set - mock will fail if any DB call was made
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should not check DB when mfaRequired=true", func(t *testing.T) {
		// Setup mock DB - no expectations should be set
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer func(db *sql.DB) {
			_ = db.Close()
		}(db)

		gormDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		require.NoError(t, err)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.False(t, claims.MFA, "Token should have MFA=false")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		// mfaRequired=true should block immediately without DB check
		handler := MFAValidate(gormDB, true)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		// No expectations set - mock will fail if any DB call was made
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
