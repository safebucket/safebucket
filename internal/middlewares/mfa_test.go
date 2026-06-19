package middlewares

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const mfaTestJWTSecret = "test-secret-key-for-mfa-testing"

func localMFARequiredProviders() configuration.Providers {
	return configuration.Providers{
		string(models.LocalProviderType): {Type: models.LocalProviderType, MFARequired: true},
	}
}

func generateRestrictedToken(secret string, user *models.User, audience string, mfaVerified bool) (string, error) {
	return helpers.NewRestrictedAccessToken(secret, user, audience, mfaVerified, nil)
}

func generateFullAccessToken(user *models.User) (string, error) {
	return helpers.NewAccessToken(mfaTestJWTSecret, user, string(models.LocalProviderType), "")
}

func newGormWithMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock, db
}

func TestMFAValidate_MFAEnforcement(t *testing.T) {
	t.Run("should require MFA setup for local user without MFA when the provider enforces it", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Audience[0])

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(
			nil,
			localMFARequiredProviders(),
		)(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"MFA_SETUP_REQUIRED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow user whose token already verified MFA", func(t *testing.T) {
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
		require.Equal(t, configuration.AudienceAccessToken, claims.Audience[0])

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should not require MFA when no provider enforcement and no enrolled device", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.Equal(t, configuration.AudienceAccessToken, claims.Audience[0])

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when MFA not required")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_RestrictedTokensSkipMFAEnforcement(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}

	t.Run("should skip MFA enforcement for restricted tokens (handled by AudienceValidate)", func(t *testing.T) {
		token, err := generateRestrictedToken(mfaTestJWTSecret, testUser, configuration.AudienceMFALogin, false)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/mfa/devices", nil)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for restricted tokens")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_NoClaims(t *testing.T) {
	t.Run("should return FORBIDDEN when no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		ctx := context.WithValue(req.Context(), AuthExcludedKey{}, true)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for excluded paths")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_NonLocalNoRequirement(t *testing.T) {
	t.Run("should allow a non-local user when the provider does not require MFA", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "oauth@example.com",
			Role:         models.RoleUser,
			ProviderType: models.OIDCProviderType,
		}

		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, "google", "")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when the provider does not require MFA")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestMFAValidate_OIDCProviderRequiresMFA(t *testing.T) {
	providers := configuration.Providers{
		"google": {Type: models.OIDCProviderType, MFARequired: true},
		"okta":   {Type: models.OIDCProviderType, MFARequired: false},
	}

	newReq := func(t *testing.T, providerKey string) *http.Request {
		t.Helper()
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "oidc@example.com",
			Role:         models.RoleUser,
			ProviderType: models.OIDCProviderType,
		}
		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, providerKey, "")
		require.NoError(t, err)
		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		return req.WithContext(context.WithValue(req.Context(), models.UserClaimKey{}, claims))
	}

	t.Run("should block OIDC user when provider enforces MFA and token lacks MFA", func(t *testing.T) {
		var nextCalled bool
		handler := MFAValidate(nil, providers)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, newReq(t, "google"))

		assert.False(t, nextCalled)
		assert.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("should skip OIDC user when provider does not enforce MFA", func(t *testing.T) {
		var nextCalled bool
		handler := MFAValidate(nil, providers)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, newReq(t, "okta"))

		assert.True(t, nextCalled)
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
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

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
		require.Equal(t, configuration.AudienceAccessToken, claims.Audience[0])
		assert.False(t, claims.MFA, "Token should have MFA=false since user had no devices at token creation")

		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(gormDB, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		assert.NoError(t, mock.ExpectationsWereMet())
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
		require.Equal(t, configuration.AudienceAccessToken, claims.Audience[0])
		assert.True(t, claims.MFA, "Token should have MFA=true since user had verified device at token creation")

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, "google", "")
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "non-local user without a provider MFA requirement is allowed")
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
		handler := MFAValidate(nil, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "MFAValidate should pass restricted tokens (audience validation is separate)")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestIsMFABypassPath_WithConfiguredRules(t *testing.T) {
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
	originalRules := configuration.MFABypassRules
	defer func() { configuration.SetMFABypassRulesForTesting(originalRules) }()

	configuration.SetMFABypassRulesForTesting([]configuration.MFABypassRule{
		{ExactPath: "/api/v1/health", Method: "GET"},
	})

	providers := localMFARequiredProviders()

	t.Run("should allow user without MFA on bypass path", func(t *testing.T) {
		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "test@example.com",
			Role:         models.RoleUser,
			ProviderType: models.LocalProviderType,
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.False(t, claims.MFA, "claims.MFA should be false for user without MFA devices")

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(nil, providers)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		}

		token, err := generateFullAccessToken(testUser)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(nil, providers)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"MFA_SETUP_REQUIRED"}}
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

		req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(nil, providers)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"MFA_SETUP_REQUIRED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})
}

func TestMFAValidate_EnrollmentCheck(t *testing.T) {
	t.Run("should block local user with MFA=false token when user has enrolled MFA", func(t *testing.T) {
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

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

		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(gormDB, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should allow local user with MFA=false token when user has no MFA enrolled", func(t *testing.T) {
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

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

		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := MFAValidate(gormDB, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when user has no MFA enrolled")
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should check enrolled devices for non-local providers", func(t *testing.T) {
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

		testUser := &models.User{
			ID:           uuid.New(),
			Email:        "oauth@example.com",
			Role:         models.RoleUser,
			ProviderType: models.OIDCProviderType,
		}

		token, err := helpers.NewAccessToken(mfaTestJWTSecret, testUser, "google", "")
		require.NoError(t, err)

		claims, err := helpers.ParseToken(mfaTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)
		require.False(t, claims.MFA)

		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "mfa_devices" WHERE user_id = $1 AND is_verified = $2`)).
			WithArgs(testUser.ID, true).
			WillReturnRows(rows)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := MFAValidate(gormDB, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should not check DB when token has MFA=true", func(t *testing.T) {
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

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
		handler := MFAValidate(gormDB, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called when token has MFA=true without DB check")
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("should not check DB when the provider enforces MFA", func(t *testing.T) {
		gormDB, mock, db := newGormWithMock(t)
		defer func(db *sql.DB) { _ = db.Close() }(db)

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

		handler := MFAValidate(
			gormDB,
			localMFARequiredProviders(),
		)(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"MFA_SETUP_REQUIRED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
