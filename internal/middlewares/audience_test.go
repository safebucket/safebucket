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

const audienceTestJWTSecret = "test-secret-key-for-audience-testing"

// TestAudienceValidate tests the AudienceValidate middleware.
func TestAudienceValidate(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}

	t.Run("should skip validation when auth is excluded", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		recorder := httptest.NewRecorder()

		// Set auth excluded flag (as Authenticate middleware would)
		ctx := context.WithValue(req.Context(), AuthExcludedKey{}, true)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for excluded paths")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should return FORBIDDEN when no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		// No claims set in context (simulates middleware chain error)
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow full access token for regular routes", func(t *testing.T) {
		token, err := helpers.NewAccessToken(audienceTestJWTSecret, testUser, string(models.LocalProviderType))
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for valid access token")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should reject restricted token for regular routes", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow restricted token for configured route", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		// MFA verify endpoint allows MFA tokens
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for valid restricted token on allowed route")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("should reject login MFA token for password reset completion", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		// Password reset completion only allows AudienceMFAReset
		req := httptest.NewRequest(
			http.MethodPost, "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete", nil,
		)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should allow password reset MFA token for password reset completion", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFAReset, true, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		// Password reset completion allows AudienceMFAReset
		req := httptest.NewRequest(
			http.MethodPost, "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete", nil,
		)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Next handler should be called for valid password reset token")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

// TestRouteAudienceValidation tests the route-level audience validation
// that prevents cross-flow attacks (e.g., login tokens accessing password reset endpoints).
func TestRouteAudienceValidation(t *testing.T) {
	t.Run("password reset completion rejects login MFA tokens", func(t *testing.T) {
		allowed := isAudienceAllowedForRoute(
			"auth:mfa:login",
			"/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			"POST",
		)
		assert.False(t, allowed, "Login MFA tokens should NOT be allowed for password reset completion")
	})

	t.Run("password reset completion accepts reset MFA tokens", func(t *testing.T) {
		allowed := isAudienceAllowedForRoute(
			"auth:mfa:password-reset",
			"/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			"POST",
		)
		assert.True(t, allowed, "Password reset MFA tokens SHOULD be allowed for password reset completion")
	})

	t.Run("MFA verify accepts login MFA tokens", func(t *testing.T) {
		allowed := isAudienceAllowedForRoute(
			"auth:mfa:login",
			"/api/v1/auth/mfa/verify",
			"POST",
		)
		assert.True(t, allowed, "Login MFA tokens SHOULD be allowed for MFA verification")
	})

	t.Run("MFA verify accepts reset MFA tokens", func(t *testing.T) {
		allowed := isAudienceAllowedForRoute(
			"auth:mfa:password-reset",
			"/api/v1/auth/mfa/verify",
			"POST",
		)
		assert.True(t, allowed, "Password reset MFA tokens SHOULD be allowed for MFA verification")
	})

	t.Run("MFA device list accepts both token types", func(t *testing.T) {
		path := "/api/v1/mfa/devices"

		loginAllowed := isAudienceAllowedForRoute("auth:mfa:login", path, "GET")
		resetAllowed := isAudienceAllowedForRoute("auth:mfa:password-reset", path, "GET")

		assert.True(t, loginAllowed, "Login MFA tokens SHOULD be allowed for device listing")
		assert.True(t, resetAllowed, "Password reset MFA tokens SHOULD be allowed for device listing")
	})

	t.Run("MFA device creation accepts both token types", func(t *testing.T) {
		path := "/api/v1/mfa/devices"

		loginAllowed := isAudienceAllowedForRoute("auth:mfa:login", path, "POST")
		resetAllowed := isAudienceAllowedForRoute("auth:mfa:password-reset", path, "POST")

		assert.True(t, loginAllowed, "Login MFA tokens SHOULD be allowed for device creation")
		assert.True(t, resetAllowed, "Password reset MFA tokens SHOULD be allowed for device creation")
	})

	t.Run("MFA device verify accepts both token types", func(t *testing.T) {
		path := "/api/v1/mfa/devices/550e8400-e29b-41d4-a716-446655440000/verify"

		loginAllowed := isAudienceAllowedForRoute("auth:mfa:login", path, "POST")
		resetAllowed := isAudienceAllowedForRoute("auth:mfa:password-reset", path, "POST")

		assert.True(t, loginAllowed, "Login MFA tokens SHOULD be allowed for device verification")
		assert.True(t, resetAllowed, "Password reset MFA tokens SHOULD be allowed for device verification")
	})

	t.Run("unconfigured route rejects all restricted tokens", func(t *testing.T) {
		// Routes without explicit rules should reject restricted tokens
		loginAllowed := isAudienceAllowedForRoute("auth:mfa:login", "/api/v1/buckets", "GET")
		resetAllowed := isAudienceAllowedForRoute("auth:mfa:password-reset", "/api/v1/buckets", "GET")

		assert.False(t, loginAllowed, "Login MFA tokens should NOT access bucket endpoints")
		assert.False(t, resetAllowed, "Password reset MFA tokens should NOT access bucket endpoints")
	})

	t.Run("wrong method is rejected", func(t *testing.T) {
		// Password reset completion only allows POST
		getNotAllowed := isAudienceAllowedForRoute(
			"auth:mfa:password-reset",
			"/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			"GET",
		)
		assert.False(t, getNotAllowed, "GET method should NOT be allowed for password reset completion")
	})

	t.Run("access token audience is rejected for restricted endpoints", func(t *testing.T) {
		// Full access tokens (app:*) should not match restricted token rules
		allowed := isAudienceAllowedForRoute(
			"app:*",
			"/api/v1/auth/mfa/verify",
			"POST",
		)
		assert.False(t, allowed, "Full access tokens should NOT match restricted endpoint rules")
	})
}

// TestGetRouteAllowedAudiences tests the helper function that returns allowed audiences for a route.
func TestGetRouteAllowedAudiences(t *testing.T) {
	t.Run("returns audiences for configured route", func(t *testing.T) {
		audiences := getRouteAllowedAudiences("/api/v1/auth/mfa/verify", "POST")
		assert.NotNil(t, audiences)
		assert.Len(t, audiences, 2)
		assert.Contains(t, audiences, "auth:mfa:login")
		assert.Contains(t, audiences, "auth:mfa:password-reset")
	})

	t.Run("returns single audience for password reset completion", func(t *testing.T) {
		audiences := getRouteAllowedAudiences(
			"/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete", "POST",
		)
		assert.NotNil(t, audiences)
		assert.Len(t, audiences, 1)
		assert.Contains(t, audiences, "auth:mfa:password-reset")
	})

	t.Run("returns nil for unconfigured route", func(t *testing.T) {
		audiences := getRouteAllowedAudiences("/api/v1/buckets", "GET")
		assert.Nil(t, audiences)
	})

	t.Run("returns nil for wrong method", func(t *testing.T) {
		audiences := getRouteAllowedAudiences("/api/v1/auth/mfa/verify", "GET")
		assert.Nil(t, audiences)
	})
}

// TestIsAudienceInList tests the helper function.
func TestIsAudienceInList(t *testing.T) {
	t.Run("returns true when audience is in list", func(t *testing.T) {
		result := isAudienceInList("auth:mfa:login", []string{"auth:mfa:login", "auth:mfa:password-reset"})
		assert.True(t, result)
	})

	t.Run("returns false when audience is not in list", func(t *testing.T) {
		result := isAudienceInList("app:*", []string{"auth:mfa:login", "auth:mfa:password-reset"})
		assert.False(t, result)
	})

	t.Run("returns false for empty list", func(t *testing.T) {
		result := isAudienceInList("auth:mfa:login", []string{})
		assert.False(t, result)
	})
}

func TestAudienceValidate_RefreshToken(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}

	t.Run("should reject refresh token for regular routes", func(t *testing.T) {
		token, err := helpers.NewRefreshToken(audienceTestJWTSecret, testUser, string(models.LocalProviderType))
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, token, false)
		require.NoError(t, err)
		assert.Equal(t, configuration.AudienceRefreshToken, claims.Aud)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should reject refresh token for MFA endpoints", func(t *testing.T) {
		token, err := helpers.NewRefreshToken(audienceTestJWTSecret, testUser, string(models.LocalProviderType))
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, token, false)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})
}

func TestAudienceValidate_UUIDPatternMatching(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}

	t.Run("should match valid UUID v4 formats in password reset", func(t *testing.T) {
		validUUIDs := []string{
			"550e8400-e29b-41d4-a716-446655440000",
			"7f3e9c10-5a2b-4d1e-8f6c-1234567890ab",
			"a1b2c3d4-e5f6-4789-abcd-ef0123456789",
		}

		for _, uuidStr := range validUUIDs {
			t.Run("UUID: "+uuidStr, func(t *testing.T) {
				token, err := helpers.NewRestrictedAccessToken(
					audienceTestJWTSecret, testUser, configuration.AudienceMFAReset, true, nil,
				)
				require.NoError(t, err)

				claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
				require.NoError(t, err)

				path := "/api/v1/auth/reset-password/" + uuidStr + "/complete"
				req := httptest.NewRequest(http.MethodPost, path, nil)
				recorder := httptest.NewRecorder()

				ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
				req = req.WithContext(ctx)

				var nextCalled bool
				handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					nextCalled = true
					w.WriteHeader(http.StatusOK)
				}))
				handler.ServeHTTP(recorder, req)

				assert.True(t, nextCalled, "Valid UUID v4 should match pattern")
				assert.Equal(t, http.StatusOK, recorder.Code)
			})
		}
	})

	t.Run("should reject invalid UUID formats", func(t *testing.T) {
		invalidCases := []struct {
			uuid   string
			reason string
		}{
			{"550e8400-e29b-11d4-a716-446655440000", "UUID v1"},
			{"550e8400-e29b-31d4-a716-446655440000", "UUID v3"},
			{"550e8400-e29b-51d4-a716-446655440000", "UUID v5"},
			{"not-a-valid-uuid-at-all-000000000000", "Invalid format"},
			{"550e8400e29b41d4a716446655440000", "No dashes"},
			{"550e8400-e29b-41d4-a716", "Too short"},
			{"550e8400-e29b-41d4-a716-446655440000-extra", "Too long"},
		}

		for _, tc := range invalidCases {
			t.Run(tc.reason, func(t *testing.T) {
				token, err := helpers.NewRestrictedAccessToken(
					audienceTestJWTSecret, testUser, configuration.AudienceMFAReset, true, nil,
				)
				require.NoError(t, err)

				claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
				require.NoError(t, err)

				path := "/api/v1/auth/reset-password/" + tc.uuid + "/complete"
				req := httptest.NewRequest(http.MethodPost, path, nil)
				recorder := httptest.NewRecorder()

				ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
				req = req.WithContext(ctx)

				handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
				handler.ServeHTTP(recorder, req)

				expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
				tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
			})
		}
	})

	t.Run("should match valid UUID v4 in MFA device verification", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		path := "/api/v1/mfa/devices/" + validUUID + "/verify"
		req := httptest.NewRequest(http.MethodPost, path, nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		var nextCalled bool
		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		assert.True(t, nextCalled, "Valid UUID should match MFA device pattern")
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestAudienceValidate_URLEdgeCases(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
	}

	t.Run("should be case-sensitive for paths", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/MFA/verify", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should reject trailing slash", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify/", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})

	t.Run("should reject double slashes", func(t *testing.T) {
		token, err := helpers.NewRestrictedAccessToken(
			audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		claims, err := helpers.ParseToken(audienceTestJWTSecret, "Bearer "+token, true)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1//auth/mfa/verify", nil)
		recorder := httptest.NewRecorder()

		ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
		req = req.WithContext(ctx)

		handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
		tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
	})
}

func TestAudienceValidate_ComprehensiveMatrix(t *testing.T) {
	testUser := &models.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		Role:         models.RoleUser,
		ProviderType: models.LocalProviderType,
		MFADevices: []models.MFADevice{
			{ID: uuid.New(), IsVerified: true},
		},
	}

	testCases := []struct {
		name        string
		audience    string
		path        string
		method      string
		shouldAllow bool
		description string
	}{
		{
			name:        "Access token on buckets",
			audience:    configuration.AudienceAccessToken,
			path:        "/api/v1/buckets",
			method:      "GET",
			shouldAllow: true,
			description: "Full access tokens work on regular endpoints",
		},
		{
			name:        "MFA login token on buckets",
			audience:    configuration.AudienceMFALogin,
			path:        "/api/v1/buckets",
			method:      "GET",
			shouldAllow: false,
			description: "Restricted tokens rejected on regular endpoints",
		},
		{
			name:        "MFA reset token on buckets",
			audience:    configuration.AudienceMFAReset,
			path:        "/api/v1/buckets",
			method:      "GET",
			shouldAllow: false,
			description: "Restricted tokens rejected on regular endpoints",
		},
		{
			name:        "Refresh token on buckets",
			audience:    configuration.AudienceRefreshToken,
			path:        "/api/v1/buckets",
			method:      "GET",
			shouldAllow: false,
			description: "Refresh tokens rejected on regular endpoints",
		},
		{
			name:        "Access token on MFA verify",
			audience:    configuration.AudienceAccessToken,
			path:        "/api/v1/auth/mfa/verify",
			method:      "POST",
			shouldAllow: false,
			description: "Full access tokens not allowed on MFA-only endpoints",
		},
		{
			name:        "MFA login token on MFA verify",
			audience:    configuration.AudienceMFALogin,
			path:        "/api/v1/auth/mfa/verify",
			method:      "POST",
			shouldAllow: true,
			description: "Login MFA tokens allowed on MFA verify",
		},
		{
			name:        "MFA reset token on MFA verify",
			audience:    configuration.AudienceMFAReset,
			path:        "/api/v1/auth/mfa/verify",
			method:      "POST",
			shouldAllow: true,
			description: "Reset MFA tokens allowed on MFA verify",
		},
		{
			name:        "MFA login token on MFA devices GET",
			audience:    configuration.AudienceMFALogin,
			path:        "/api/v1/mfa/devices",
			method:      "GET",
			shouldAllow: true,
			description: "Login MFA tokens allowed on device list",
		},
		{
			name:        "MFA reset token on MFA devices POST",
			audience:    configuration.AudienceMFAReset,
			path:        "/api/v1/mfa/devices",
			method:      "POST",
			shouldAllow: true,
			description: "Reset MFA tokens allowed on device creation",
		},
		{
			name:        "MFA login token on password reset complete",
			audience:    configuration.AudienceMFALogin,
			path:        "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			method:      "POST",
			shouldAllow: false,
			description: "Login tokens NOT allowed on password reset completion",
		},
		{
			name:        "MFA reset token on password reset complete",
			audience:    configuration.AudienceMFAReset,
			path:        "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			method:      "POST",
			shouldAllow: true,
			description: "Reset tokens allowed on password reset completion",
		},
		{
			name:        "Access token on password reset complete",
			audience:    configuration.AudienceAccessToken,
			path:        "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/complete",
			method:      "POST",
			shouldAllow: false,
			description: "Full access tokens not allowed on password reset completion",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var token string
			var err error

			switch tc.audience {
			case configuration.AudienceAccessToken:
				token, err = helpers.NewAccessToken(audienceTestJWTSecret, testUser, string(models.LocalProviderType))
			case configuration.AudienceMFALogin:
				token, err = helpers.NewRestrictedAccessToken(
					audienceTestJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
				)
			case configuration.AudienceMFAReset:
				token, err = helpers.NewRestrictedAccessToken(
					audienceTestJWTSecret, testUser, configuration.AudienceMFAReset, true, nil,
				)
			case configuration.AudienceRefreshToken:
				token, err = helpers.NewRefreshToken(audienceTestJWTSecret, testUser, string(models.LocalProviderType))
			}
			require.NoError(t, err)

			requireBearer := tc.audience != configuration.AudienceRefreshToken
			if requireBearer {
				token = "Bearer " + token
			}

			claims, err := helpers.ParseToken(audienceTestJWTSecret, token, requireBearer)
			require.NoError(t, err)

			req := httptest.NewRequest(tc.method, tc.path, nil)
			recorder := httptest.NewRecorder()

			ctx := context.WithValue(req.Context(), models.UserClaimKey{}, claims)
			req = req.WithContext(ctx)

			var nextCalled bool
			handler := AudienceValidate(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			handler.ServeHTTP(recorder, req)

			if tc.shouldAllow {
				assert.True(t, nextCalled, tc.description)
				assert.Equal(t, http.StatusOK, recorder.Code)
			} else {
				assert.False(t, nextCalled, tc.description)
				assert.Equal(t, http.StatusForbidden, recorder.Code)
			}
		})
	}
}
