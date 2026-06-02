package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-for-testing"

func mockAuthenticatedNextHandler(w http.ResponseWriter, r *http.Request) {
	userClaims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("NO_CLAIMS"))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK:" + userClaims.Email))
}

func generateTestToken(secret string, user *models.User, expiresIn time.Duration) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Provider: "test",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    configuration.AppName,
			Audience:  jwt.ClaimStrings{"app:*"},
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(expiresIn)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

//nolint:unparam // test helper kept stable for callers; underlying token always uses default expiry.
func generateTestTokenWithSession(
	secret string, user *models.User, _ time.Duration, mc *cache.MemoryCache,
) (string, error) {
	sid := uuid.New().String()
	if err := cache.CreateSession(mc, user.ID.String(), sid); err != nil {
		return "", err
	}
	return generateTestTokenWithSID(secret, user, 0, sid)
}

func TestAuthenticate(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	testUser := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	validToken, err := generateTestTokenWithSession(testJWTSecret, testUser, time.Hour, mc)
	require.NoError(t, err)

	expiredToken, err := generateTestToken(testJWTSecret, testUser, -time.Hour)
	require.NoError(t, err)

	wrongSecretToken, err := generateTestToken("wrong-secret", testUser, time.Hour)
	require.NoError(t, err)

	testCases := []struct {
		name               string
		authHeader         string
		path               string
		method             string
		expectedStatus     int
		expectedBody       string
		shouldHaveClaims   bool
		expectedClaimEmail string
	}{
		{
			name:               "Valid JWT token with Bearer prefix",
			authHeader:         "Bearer " + validToken,
			path:               "/api/v1/buckets",
			method:             http.MethodGet,
			expectedStatus:     http.StatusOK,
			expectedBody:       "OK:test@example.com",
			shouldHaveClaims:   true,
			expectedClaimEmail: "test@example.com",
		},
		{
			name:           "Missing Authorization header",
			authHeader:     "",
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name:           "Empty Authorization header",
			authHeader:     "Bearer ",
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name:           "Invalid JWT token (malformed)",
			authHeader:     "Bearer invalid.token.here",
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name:           "JWT without Bearer prefix",
			authHeader:     validToken,
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name:           "JWT with wrong secret",
			authHeader:     "Bearer " + wrongSecretToken,
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
		{
			name:           "Expired JWT token",
			authHeader:     "Bearer " + expiredToken,
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
			expectedBody:   "",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			recorder := httptest.NewRecorder()

			handler := Authenticate(testJWTSecret, mc, 600)(
				http.HandlerFunc(mockAuthenticatedNextHandler),
			)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusForbidden {
				expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
				tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
			} else if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, recorder.Body.String())
			}
		})
	}
}

func TestAuthenticate_ExcludedPaths(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	testUser := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	validToken, err := generateTestTokenWithSession(testJWTSecret, testUser, time.Hour, mc)
	require.NoError(t, err)

	testCases := []struct {
		name           string
		path           string
		method         string
		authHeader     string
		expectedStatus int
		description    string
	}{
		{
			name:           "Excluded path - /api/v1/auth/* without token",
			path:           "/api/v1/auth/login",
			method:         http.MethodPost,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Auth endpoints should not require authentication",
		},
		{
			name:           "Excluded path - /api/v1/auth/providers without token",
			path:           "/api/v1/auth/providers",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Auth provider list should be public",
		},
		{
			name:           "Excluded path - /api/v1/auth/providers/*/login without token (POST)",
			path:           "/api/v1/auth/providers/myldap/login",
			method:         http.MethodPost,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Provider login (POST) should not require authentication",
		},
		{
			name:           "Excluded path - /api/v1/invites/*/challenges without token (POST)",
			path:           "/api/v1/invites/550e8400-e29b-41d4-a716-446655440000/challenges",
			method:         http.MethodPost,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Invite challenge endpoints (POST) should not require auth",
		},
		{
			name:           "Excluded path - /api/v1/invites/*/challenges/*/validate without token (POST)",
			path:           "/api/v1/invites/550e8400-e29b-41d4-a716-446655440000/challenges/660e8400-e29b-41d4-a716-446655440000/validate",
			method:         http.MethodPost,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Invite challenge validate endpoint (POST) should not require auth",
		},
		{
			name:           "Required path - /api/v1/buckets without token",
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusForbidden,
			description:    "Bucket endpoints require authentication",
		},
		{
			name:           "Required path - /api/v1/users without token",
			path:           "/api/v1/users",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusForbidden,
			description:    "User endpoints require authentication",
		},
		{
			name:           "Required path - /api/v1/buckets with valid token",
			path:           "/api/v1/buckets",
			method:         http.MethodGet,
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
			description:    "Valid token should pass authentication",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			recorder := httptest.NewRecorder()

			simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			handler := Authenticate(testJWTSecret, mc, 600)(simpleHandler)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code, tt.description)

			if tt.expectedStatus == http.StatusForbidden {
				expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
				tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
			}
		})
	}
}

func TestIsAuthExcluded(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{
			name:     "Excluded - /api/v1/auth/login with POST",
			path:     "/api/v1/auth/login",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/auth/login with GET",
			path:     "/api/v1/auth/login",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Excluded - /api/v1/auth/providers with GET",
			path:     "/api/v1/auth/providers",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/providers/google/begin with GET",
			path:     "/api/v1/auth/providers/google/begin",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/providers/myldap/login with POST",
			path:     "/api/v1/auth/providers/myldap/login",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/auth/providers/myldap/login with GET (method mismatch)",
			path:     "/api/v1/auth/providers/myldap/login",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Excluded - /api/v1/auth/verify with POST",
			path:     "/api/v1/auth/verify",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/refresh with POST",
			path:     "/api/v1/auth/refresh",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/auth/mfa/verify requires auth",
			path:     "/api/v1/auth/mfa/verify",
			method:   "POST",
			expected: false,
		},
		{
			name:     "Excluded - /api/v1/auth/reset-password with POST (initiate)",
			path:     "/api/v1/auth/reset-password",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/reset-password/{id}/validate with POST",
			path:     "/api/v1/auth/reset-password/550e8400-e29b-41d4-a716-446655440000/validate",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/auth/reset-password/{id}/complete requires auth",
			path:     "/api/v1/auth/reset-password/abc-123/complete",
			method:   "POST",
			expected: false,
		},
		{
			name:     "Excluded - /api/v1/invites/123/challenges with POST",
			path:     "/api/v1/invites/550e8400-e29b-41d4-a716-446655440000/challenges",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/invites/*/challenges/*/validate with POST",
			path:     "/api/v1/invites/550e8400-e29b-41d4-a716-446655440000/challenges/660e8400-e29b-41d4-a716-446655440000/validate",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/invites/*/challenges with GET (method mismatch)",
			path:     "/api/v1/invites/550e8400-e29b-41d4-a716-446655440000/challenges",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/invites with POST",
			path:     "/api/v1/invites",
			method:   "POST",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/buckets (auth required by default)",
			path:     "/api/v1/buckets",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/users (auth required by default)",
			path:     "/api/v1/users",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/buckets with POST",
			path:     "/api/v1/buckets",
			method:   "POST",
			expected: false,
		},
		{
			name:     "Not excluded - random path with no rules",
			path:     "/api/v1/random",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - root path",
			path:     "/",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/mfa requires auth",
			path:     "/api/v1/mfa/devices",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Excluded - /api/v1/auth/logout with POST",
			path:     "/api/v1/auth/logout",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/providers/ trailing slash with GET",
			path:     "/api/v1/auth/providers/",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/auth/providers/google/callback with GET",
			path:     "/api/v1/auth/providers/google/callback",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id} with GET",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id}/ trailing slash with GET",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000/",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id}/auth with POST",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000/auth",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id}/files with POST",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000/files",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id}/files/{fileId} with GET",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000/files/660e8400-e29b-41d4-a716-446655440000",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - /api/v1/shares/{id}/files/{fileId} with PATCH",
			path:     "/api/v1/shares/550e8400-e29b-41d4-a716-446655440000/files/660e8400-e29b-41d4-a716-446655440000",
			method:   "PATCH",
			expected: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := isPathExcludedFromAuth(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAuthExcluded_AdversarialInputs(t *testing.T) {
	const shareID = "550e8400-e29b-41d4-a716-446655440000"

	testCases := []struct {
		name   string
		path   string
		method string
	}{
		{"traversal out of shares prefix", "/api/v1/shares/" + shareID + "/../../buckets/" + shareID, "GET"},
		{"traversal out of providers prefix", "/api/v1/auth/providers/../../buckets", "GET"},
		{
			"traversal from reset-password to complete",
			"/api/v1/auth/reset-password/" + shareID + "/../complete",
			"POST",
		},
		{"traversal in shares files segment", "/api/v1/shares/" + shareID + "/files/../../buckets", "GET"},

		{"percent-encoded dot-dot in shares", "/api/v1/shares/" + shareID + "/%2e%2e/buckets", "GET"},
		{"percent-encoded slash traversal in providers", "/api/v1/auth/providers/..%2f..%2fbuckets", "GET"},
		{"percent-encoded shares id", "/api/v1/shares/%2e%2e", "GET"},

		{"leading double slash on login", "//api/v1/auth/login", "POST"},
		{"embedded double slash on login", "/api/v1//auth/login", "POST"},
		{"trailing slash on login exact path", "/api/v1/auth/login/", "POST"},

		{"providers prefix over-match", "/api/v1/auth/providersEVIL", "GET"},
		{"shares uuid suffix over-match", "/api/v1/shares/" + shareID + "extra", "GET"},
		{"shares files suffix over-match", "/api/v1/shares/" + shareID + "/filesEVIL", "GET"},

		{"providers list with POST", "/api/v1/auth/providers", "POST"},
		{"providers list with DELETE", "/api/v1/auth/providers", "DELETE"},
		{"providers begin with POST", "/api/v1/auth/providers/google/begin", "POST"},

		{"non-uuid share id", "/api/v1/shares/not-a-uuid/auth", "POST"},
		{"non-uuid invite id", "/api/v1/invites/not-a-uuid/challenges", "POST"},
		{"non-uuid reset-password id", "/api/v1/auth/reset-password/not-a-uuid/validate", "POST"},
		{"uppercase uuid share id", "/api/v1/shares/550E8400-E29B-41D4-A716-446655440000", "GET"},

		{"unicode lookalike slash in login", "/api/v1/auth／login", "POST"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, isPathExcludedFromAuth(tt.path, tt.method),
				"path %q (%s) must require authentication", tt.path, tt.method)
		})
	}
}

func TestAuthenticate_UserClaimsInContext(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	testUser := &models.User{
		ID:    uuid.New(),
		Email: "context-test@example.com",
		Role:  models.RoleAdmin,
	}

	validToken, err := generateTestTokenWithSession(testJWTSecret, testUser, time.Hour, mc)
	require.NoError(t, err)

	var capturedClaims models.UserClaims
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		require.True(t, ok, "UserClaims should be in context")
		capturedClaims = claims
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	recorder := httptest.NewRecorder()

	handler := Authenticate(testJWTSecret, mc, 600)(testHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, testUser.Email, capturedClaims.Email)
	assert.Equal(t, testUser.ID, capturedClaims.UserID)
	assert.Equal(t, testUser.Role, capturedClaims.Role)
	assert.Equal(t, "local", capturedClaims.Provider)
	assert.Equal(t, configuration.AppName, capturedClaims.Issuer)
}

func generateTestTokenWithSID(
	secret string, user *models.User, _ time.Duration, sid string,
) (string, error) {
	return helpers.NewAccessToken(secret, user, "local", sid)
}

func TestAuthenticate_SessionRevocation(t *testing.T) {
	const refreshTokenExpiry = 600

	testUser := &models.User{
		ID:    uuid.New(),
		Email: "session@example.com",
		Role:  models.RoleUser,
	}

	t.Run("Active session passes", func(t *testing.T) {
		mc := cache.NewMemoryCache()
		t.Cleanup(func() { mc.Close() })

		sid := uuid.New().String()
		require.NoError(t, cache.CreateSession(mc, testUser.ID.String(), sid))

		token, err := generateTestTokenWithSID(testJWTSecret, testUser, time.Hour, sid)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler := Authenticate(testJWTSecret, mc, refreshTokenExpiry)(
			http.HandlerFunc(mockAuthenticatedNextHandler),
		)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "OK:session@example.com")
	})

	t.Run("Revoked session blocked", func(t *testing.T) {
		mc := cache.NewMemoryCache()
		t.Cleanup(func() { mc.Close() })

		sid := uuid.New().String()
		require.NoError(t, cache.CreateSession(mc, testUser.ID.String(), sid))
		require.NoError(t, cache.RevokeSession(mc, testUser.ID.String(), sid))

		token, err := generateTestTokenWithSID(testJWTSecret, testUser, time.Hour, sid)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler := Authenticate(testJWTSecret, mc, refreshTokenExpiry)(
			http.HandlerFunc(mockAuthenticatedNextHandler),
		)
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusUnauthorized, Error: []string{"SESSION_REVOKED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusUnauthorized, expected)
	})

	t.Run("Unknown SID blocked", func(t *testing.T) {
		mc := cache.NewMemoryCache()
		t.Cleanup(func() { mc.Close() })

		sid := uuid.New().String()
		token, err := generateTestTokenWithSID(testJWTSecret, testUser, time.Hour, sid)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler := Authenticate(testJWTSecret, mc, refreshTokenExpiry)(
			http.HandlerFunc(mockAuthenticatedNextHandler),
		)
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusUnauthorized, Error: []string{"SESSION_REVOKED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusUnauthorized, expected)
	})

	t.Run("Restricted token without SID passes", func(t *testing.T) {
		mc := cache.NewMemoryCache()
		t.Cleanup(func() { mc.Close() })

		token, err := helpers.NewRestrictedAccessToken(
			testJWTSecret, testUser, configuration.AudienceMFALogin, false, nil,
		)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler := Authenticate(testJWTSecret, mc, refreshTokenExpiry)(
			http.HandlerFunc(mockAuthenticatedNextHandler),
		)
		handler.ServeHTTP(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "OK:session@example.com")
	})

	t.Run("No SID rejects token", func(t *testing.T) {
		mc := cache.NewMemoryCache()
		t.Cleanup(func() { mc.Close() })

		// NewAccessToken with empty SID produces a valid JWE-wrapped token that fails session check.
		token, err := helpers.NewAccessToken(testJWTSecret, testUser, "local", "")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()

		handler := Authenticate(testJWTSecret, mc, refreshTokenExpiry)(
			http.HandlerFunc(mockAuthenticatedNextHandler),
		)
		handler.ServeHTTP(recorder, req)

		expected := models.Error{Status: http.StatusUnauthorized, Error: []string{"SESSION_REVOKED"}}
		tests.AssertJSONResponse(t, recorder, http.StatusUnauthorized, expected)
	})
}

func TestAuthenticate_ContextPropagation(t *testing.T) {
	testUser := &models.User{
		ID:    uuid.New(),
		Email: "propagation@example.com",
		Role:  models.RoleUser,
	}

	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	sid := uuid.New().String()
	require.NoError(t, cache.CreateSession(mc, testUser.ID.String(), sid))

	validToken, err := generateTestTokenWithSID(testJWTSecret, testUser, time.Hour, sid)
	require.NoError(t, err)

	type testContextKey struct{}
	existingValue := "existing-context-value"

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.Context().Value(testContextKey{})
		assert.Equal(t, existingValue, val, "Existing context values should be preserved")

		claims, ok := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		assert.True(t, ok, "UserClaims should be added to context")
		assert.Equal(t, testUser.Email, claims.Email)

		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)

	ctx := context.WithValue(req.Context(), testContextKey{}, existingValue)
	req = req.WithContext(ctx)

	recorder := httptest.NewRecorder()

	handler := Authenticate(testJWTSecret, mc, 600)(testHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestAuthenticate_BearerHeaderWinsOverCookie(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	cookieUser := &models.User{ID: uuid.New(), Email: "cookie@example.com", Role: models.RoleUser}
	headerUser := &models.User{ID: uuid.New(), Email: "header@example.com", Role: models.RoleUser}

	cookieToken, err := generateTestTokenWithSession(testJWTSecret, cookieUser, time.Hour, mc)
	require.NoError(t, err)
	headerToken, err := generateTestTokenWithSession(testJWTSecret, headerUser, time.Hour, mc)
	require.NoError(t, err)

	var captured models.UserClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		captured = c
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: cookieToken})
	req.Header.Set("Authorization", "Bearer "+headerToken)

	recorder := httptest.NewRecorder()
	Authenticate(testJWTSecret, mc, 600)(next).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, headerUser.Email, captured.Email,
		"Authorization header must take precedence over cookie")
}

func TestAuthenticate_MFACookieWinsOverAccessCookie(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	staleUser := &models.User{ID: uuid.New(), Email: "stale@example.com", Role: models.RoleUser}
	mfaUser := &models.User{ID: uuid.New(), Email: "mfa@example.com", Role: models.RoleUser}

	staleAccessToken, err := generateTestTokenWithSession(testJWTSecret, staleUser, time.Hour, mc)
	require.NoError(t, err)
	mfaToken, err := helpers.NewRestrictedAccessToken(
		testJWTSecret, mfaUser, configuration.AudienceMFALogin, false, nil,
	)
	require.NoError(t, err)

	var captured models.UserClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		captured = c
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/mfa/verify", nil)
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: staleAccessToken})
	req.AddCookie(&http.Cookie{Name: "safebucket_mfa_token", Value: mfaToken})

	recorder := httptest.NewRecorder()
	Authenticate(testJWTSecret, mc, 600)(next).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, mfaUser.Email, captured.Email,
		"MFA cookie must take precedence over access cookie")
}

func TestAuthenticate_CookieFallback(t *testing.T) {
	mc := cache.NewMemoryCache()
	t.Cleanup(func() { mc.Close() })

	user := &models.User{ID: uuid.New(), Email: "cookie@example.com", Role: models.RoleUser}
	token, err := generateTestTokenWithSession(testJWTSecret, user, time.Hour, mc)
	require.NoError(t, err)

	var captured models.UserClaims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Context().Value(models.UserClaimKey{}).(models.UserClaims)
		captured = c
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/buckets", nil)
	req.AddCookie(&http.Cookie{Name: "safebucket_access_token", Value: token})

	recorder := httptest.NewRecorder()
	Authenticate(testJWTSecret, mc, 600)(next).ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, user.Email, captured.Email)
}
