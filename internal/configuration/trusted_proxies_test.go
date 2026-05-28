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
		{name: "valid CIDR and IP mix", entries: []string{"10.0.0.0/8", "127.0.0.1", "::1"}},
		{name: "empty slice rejected", entries: nil, wantErr: true},
		{name: "garbage element rejected", entries: []string{"10.0.0.0/8", "not-an-ip"}, wantErr: true},
		{name: "empty element rejected", entries: []string{"10.0.0.1", ""}, wantErr: true},
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
		wantErr   bool
	}{
		{name: "valid CIDR", entries: []string{"10.0.0.0/8"}, wantCount: 1},
		{name: "bare IPv4 normalised to /32", entries: []string{"192.168.1.1"}, wantCount: 1},
		{name: "bare IPv6 normalised to /128", entries: []string{"::1"}, wantCount: 1},
		{name: "mixed entries", entries: []string{"10.0.0.0/8", "127.0.0.1", "fd00::/8"}, wantCount: 3},
		{name: "empty entries skipped", entries: []string{"", "  ", "10.0.0.1"}, wantCount: 1},
		{name: "nil", entries: nil, wantCount: 0},
		{name: "malformed CIDR rejected", entries: []string{"10.0.0.0/99"}, wantErr: true},
		{name: "garbage rejected", entries: []string{"not-an-ip"}, wantErr: true},
		{name: "partial address rejected", entries: []string{"10.0.0"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nets, err := ParseTrustedProxies(tt.entries)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %v, got none", tt.entries)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nets) != tt.wantCount {
				t.Fatalf("got %d networks, want %d", len(nets), tt.wantCount)
			}
		})
	}
}
