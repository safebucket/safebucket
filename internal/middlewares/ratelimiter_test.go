package middlewares

import (
	"net"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/configuration"
)

func mustParseProxies(_ *testing.T, entries []string) []*net.IPNet {
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
