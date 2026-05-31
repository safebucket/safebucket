package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestRateLimitUnauthenticated(t *testing.T) {
	testCases := []struct {
		name           string
		clientInfo     *models.ClientInfo
		expectedStatus int
		expectNext     bool
	}{
		{
			name:           "Missing client info returns 500",
			clientInfo:     nil,
			expectedStatus: http.StatusInternalServerError,
			expectNext:     false,
		},
		{
			name:           "Empty client IP returns 500",
			clientInfo:     &models.ClientInfo{IP: "", UserAgent: "test-agent/1.0"},
			expectedStatus: http.StatusInternalServerError,
			expectNext:     false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.clientInfo != nil {
				ctx := context.WithValue(req.Context(), models.ClientInfoKey{}, *tt.clientInfo)
				req = req.WithContext(ctx)
			}
			recorder := httptest.NewRecorder()

			var nextCalled bool
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RateLimit(cache.NewMemoryCache(), 200, 20)(next)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
			assert.Equal(t, tt.expectNext, nextCalled)
		})
	}
}
