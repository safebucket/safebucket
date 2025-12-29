package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api/internal/models"
	"api/internal/tests"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-for-testing"

// mockAuthenticatedNextHandler checks if UserClaims are in context.
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

// generateTestToken creates a valid JWT token for testing.
func generateTestToken(secret string, user *models.User, expiresIn time.Duration) (string, error) {
	claims := models.UserClaims{
		Email:    user.Email,
		UserID:   user.ID,
		Role:     user.Role,
		Aud:      "app:*",
		Provider: "test",
		Issuer:   "safebucket",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  &jwt.NumericDate{Time: time.Now()},
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(expiresIn)},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func TestAuthenticate(t *testing.T) {
	testUser := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	validToken, err := generateTestToken(testJWTSecret, testUser, time.Hour)
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

			handler := Authenticate(testJWTSecret, false)(http.HandlerFunc(mockAuthenticatedNextHandler))
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
	testUser := &models.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Role:  models.RoleUser,
	}

	validToken, err := generateTestToken(testJWTSecret, testUser, time.Hour)
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
			name:           "Excluded path - /api/v1/invites/* without token (GET)",
			path:           "/api/v1/invites/123",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusOK,
			description:    "Invite endpoints (GET) should not require auth",
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

			// For excluded paths, use a simple handler that just returns OK
			simpleHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			})

			handler := Authenticate(testJWTSecret, false)(simpleHandler)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code, tt.description)

			if tt.expectedStatus == http.StatusForbidden {
				expected := models.Error{Status: http.StatusForbidden, Error: []string{"FORBIDDEN"}}
				tests.AssertJSONResponse(t, recorder, http.StatusForbidden, expected)
			}
		})
	}
}

func TestIsExcluded(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		method   string
		expected bool
	}{
		{
			name:     "Excluded - prefix match /api/v1/auth with GET",
			path:     "/api/v1/auth/login",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Excluded - prefix match /api/v1/auth with POST",
			path:     "/api/v1/auth/login",
			method:   "POST",
			expected: true,
		},
		{
			name:     "Excluded - prefix match /api/v1/auth with wildcard",
			path:     "/api/v1/auth/providers",
			method:   "PUT",
			expected: true,
		},
		{
			name:     "Excluded - prefix match /api/v1/invites with GET",
			path:     "/api/v1/invites/123",
			method:   "GET",
			expected: true,
		},
		{
			name:     "Not excluded - /api/v1/buckets (RequireAuth: true)",
			path:     "/api/v1/buckets",
			method:   "GET",
			expected: false,
		},
		{
			name:     "Not excluded - /api/v1/users (RequireAuth: true)",
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
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := isExcluded(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthenticate_UserClaimsInContext(t *testing.T) {
	testUser := &models.User{
		ID:    uuid.New(),
		Email: "context-test@example.com",
		Role:  models.RoleAdmin,
	}

	validToken, err := generateTestToken(testJWTSecret, testUser, time.Hour)
	require.NoError(t, err)

	// Handler that extracts and validates claims from context
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

	handler := Authenticate(testJWTSecret, false)(testHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, testUser.Email, capturedClaims.Email)
	assert.Equal(t, testUser.ID, capturedClaims.UserID)
	assert.Equal(t, testUser.Role, capturedClaims.Role)
	assert.Equal(t, "test", capturedClaims.Provider)
	assert.Equal(t, "safebucket", capturedClaims.Issuer)
}

func TestAuthenticate_ContextPropagation(t *testing.T) {
	testUser := &models.User{
		ID:    uuid.New(),
		Email: "propagation@example.com",
		Role:  models.RoleUser,
	}

	validToken, err := generateTestToken(testJWTSecret, testUser, time.Hour)
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

	handler := Authenticate(testJWTSecret, false)(testHandler)
	handler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}
