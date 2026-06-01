package middlewares

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
)

func mustParseProxies(t *testing.T, entries []string) []*net.IPNet {
	t.Helper()
	return configuration.ParseTrustedProxies(entries)
}

func newRequest(t *testing.T, remoteAddr, xff string) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	r.RemoteAddr = remoteAddr
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xff            string
		trustedProxies []string
		want           string
	}{
		{
			name:           "untrusted peer ignores X-Forwarded-For",
			remoteAddr:     "203.0.113.10:54321",
			xff:            "1.2.3.4",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "203.0.113.10",
		},
		{
			name:           "trusted CIDR peer honours X-Forwarded-For",
			remoteAddr:     "10.1.2.3:443",
			xff:            "1.2.3.4",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "1.2.3.4",
		},
		{
			name:           "trusted /32 peer honours X-Forwarded-For",
			remoteAddr:     "10.1.2.3:443",
			xff:            "1.2.3.4",
			trustedProxies: []string{"10.1.2.3/32"},
			want:           "1.2.3.4",
		},
		{
			name:           "leftmost spoof is ignored, rightmost untrusted hop wins",
			remoteAddr:     "10.0.0.1:443",
			xff:            "6.6.6.6, 1.2.3.4",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "1.2.3.4",
		},
		{
			name:           "skips trailing trusted hops to the rightmost untrusted",
			remoteAddr:     "10.0.0.1:443",
			xff:            "1.2.3.4, 10.0.0.5, 10.0.0.6",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "1.2.3.4",
		},
		{
			name:           "no trusted proxies configured uses RemoteAddr",
			remoteAddr:     "203.0.113.10:54321",
			xff:            "1.2.3.4",
			trustedProxies: nil,
			want:           "203.0.113.10",
		},
		{
			name:           "trusted peer with no XFF falls back to RemoteAddr",
			remoteAddr:     "10.0.0.1:443",
			xff:            "",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "10.0.0.1",
		},
		{
			name:           "all XFF hops trusted falls back to RemoteAddr",
			remoteAddr:     "10.0.0.1:443",
			xff:            "10.0.0.2, 10.0.0.3",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "10.0.0.1",
		},
		{
			name:           "only last proxy trusted: stops at first untrusted hop from the right",
			remoteAddr:     "201.101.101.201:443",
			xff:            "203.0.113.195, 101.101.101.102, 201.101.101.102",
			trustedProxies: []string{"201.0.0.0/8"},
			want:           "101.101.101.102",
		},
		{
			name:           "two trusted ranges: skips both and returns real client IP",
			remoteAddr:     "201.101.101.201:443",
			xff:            "203.0.113.195, 101.101.101.102, 201.101.101.102",
			trustedProxies: []string{"101.0.0.0/8", "201.0.0.0/8"},
			want:           "203.0.113.195",
		},
		{
			name:           "invalid hop in XFF is skipped, next valid untrusted hop wins",
			remoteAddr:     "10.0.0.1:443",
			xff:            "203.0.113.195, not-an-ip, 10.0.0.2",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "203.0.113.195",
		},
		{
			name:           "all XFF hops invalid or trusted falls back to RemoteAddr",
			remoteAddr:     "10.0.0.1:443",
			xff:            "not-an-ip, 10.0.0.2",
			trustedProxies: []string{"10.0.0.0/8"},
			want:           "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRequest(t, tt.remoteAddr, tt.xff)
			got, err := getClientIP(r, mustParseProxies(t, tt.trustedProxies))
			if err != nil {
				t.Fatalf("getClientIP returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("getClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}

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
			name:              "Trusted CIDR proxy returns rightmost untrusted XFF hop",
			remoteAddr:        "10.0.0.9:4242",
			xForwardedFor:     "198.51.100.7, 10.0.0.1",
			userAgent:         "proxied-agent/2.0",
			trustedProxies:    []string{"10.0.0.0/8"},
			expectedIP:        "198.51.100.7",
			expectedUserAgent: "proxied-agent/2.0",
		},
		{
			name:              "Untrusted source ignores spoofed X-Forwarded-For",
			remoteAddr:        "203.0.113.5:4242",
			xForwardedFor:     "1.2.3.4",
			userAgent:         "spoofer/1.0",
			trustedProxies:    []string{"10.0.0.0/8"},
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
				info, ok := r.Context().Value(models.ClientInfoKey{}).(models.ClientInfo)
				assert.True(t, ok)
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
