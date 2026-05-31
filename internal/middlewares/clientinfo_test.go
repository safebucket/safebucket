package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safebucket/safebucket/internal/helpers"

	"github.com/stretchr/testify/assert"
)

func TestClientInfoMiddleware(t *testing.T) {
	testCases := []struct {
		name              string
		remoteAddr        string
		xForwardedFor     string
		userAgent         string
		trustedProxies    []string
		expectedIP        string
		expectedUserAgent string
	}{
		{
			name:              "No trusted proxies uses RemoteAddr",
			remoteAddr:        "203.0.113.5:4242",
			xForwardedFor:     "10.0.0.1",
			userAgent:         "test-agent/1.0",
			trustedProxies:    nil,
			expectedIP:        "203.0.113.5",
			expectedUserAgent: "test-agent/1.0",
		},
		{
			name:              "Trusted proxy uses leftmost X-Forwarded-For hop",
			remoteAddr:        "10.0.0.9:4242",
			xForwardedFor:     "198.51.100.7, 10.0.0.1",
			userAgent:         "proxied-agent/2.0",
			trustedProxies:    []string{"10.0.0.9"},
			expectedIP:        "198.51.100.7",
			expectedUserAgent: "proxied-agent/2.0",
		},
		{
			name:              "Untrusted source ignores spoofed X-Forwarded-For",
			remoteAddr:        "203.0.113.5:4242",
			xForwardedFor:     "1.2.3.4",
			userAgent:         "spoofer/1.0",
			trustedProxies:    []string{"10.0.0.9"},
			expectedIP:        "203.0.113.5",
			expectedUserAgent: "spoofer/1.0",
		},
		{
			name:              "Bare IP RemoteAddr fallback",
			remoteAddr:        "203.0.113.5",
			userAgent:         "bare-agent/1.0",
			trustedProxies:    nil,
			expectedIP:        "203.0.113.5",
			expectedUserAgent: "bare-agent/1.0",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			req.Header.Set("User-Agent", tt.userAgent)
			recorder := httptest.NewRecorder()

			var captured bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				info, err := helpers.GetClientInfo(r.Context())
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedIP, info.IP)
				assert.Equal(t, tt.expectedUserAgent, info.UserAgent)
				captured = true
				w.WriteHeader(http.StatusOK)
			})

			handler := ClientInfo(tt.trustedProxies)(next)
			handler.ServeHTTP(recorder, req)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.True(t, captured, "next handler should have been called")
		})
	}
}
