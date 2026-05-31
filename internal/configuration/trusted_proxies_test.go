package configuration

import (
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/safebucket/safebucket/internal/models"
)

func TestTrustedProxiesValidationRule(t *testing.T) {
	validate := validator.New()
	tests := []struct {
		name    string
		entries []string
		wantErr bool
	}{
		{name: "valid CIDR list", entries: []string{"10.0.0.0/8", "127.0.0.1/32", "::1/128"}},
		{name: "empty slice allowed", entries: nil},
		{name: "garbage element rejected", entries: []string{"10.0.0.0/8", "not-an-ip"}, wantErr: true},
		{name: "bare IPv4 rejected", entries: []string{"127.0.0.1"}, wantErr: true},
		{name: "bare IPv6 rejected", entries: []string{"::1"}, wantErr: true},
		{name: "empty element rejected", entries: []string{"10.0.0.1/32", ""}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := models.AppConfiguration{TrustedProxies: tt.entries}
			err := validate.StructPartial(app, "TrustedProxies")
			if (err != nil) != tt.wantErr {
				t.Fatalf("StructPartial(%v) error = %v, wantErr %v", tt.entries, err, tt.wantErr)
			}
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	tests := []struct {
		name      string
		entries   []string
		wantCount int
	}{
		{name: "valid CIDR", entries: []string{"10.0.0.0/8"}, wantCount: 1},
		{name: "single host IPv4 CIDR", entries: []string{"192.168.1.1/32"}, wantCount: 1},
		{name: "single host IPv6 CIDR", entries: []string{"::1/128"}, wantCount: 1},
		{name: "mixed CIDR entries", entries: []string{"10.0.0.0/8", "127.0.0.1/32", "fd00::/8"}, wantCount: 3},
		{name: "empty entries skipped", entries: []string{"", "  ", "10.0.0.1/32"}, wantCount: 1},
		{name: "nil", entries: nil, wantCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets := ParseTrustedProxies(tt.entries)
			if len(nets) != tt.wantCount {
				t.Fatalf("got %d networks, want %d", len(nets), tt.wantCount)
			}
		})
	}
}
