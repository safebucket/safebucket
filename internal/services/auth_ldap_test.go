package services

import (
	"testing"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitDisplayName(t *testing.T) {
	cases := []struct {
		in         string
		wantFirst  string
		wantLast   string
		descriptor string
	}{
		{"", "", "", "empty"},
		{"   ", "", "", "whitespace only"},
		{"Madonna", "Madonna", "", "single token"},
		{"Jane Doe", "Jane", "Doe", "first last"},
		{"Jane  Doe", "Jane", "Doe", "double space collapses via TrimSpace on last"},
		{"  Jane Doe  ", "Jane", "Doe", "outer whitespace trimmed"},
		{"Jane Doe Smith", "Jane", "Doe Smith", "compound last name"},
	}
	for _, tc := range cases {
		t.Run(tc.descriptor, func(t *testing.T) {
			first, last := splitDisplayName(tc.in)
			assert.Equal(t, tc.wantFirst, first)
			assert.Equal(t, tc.wantLast, last)
		})
	}
}

func TestResolveCredentialProvider(t *testing.T) {
	t.Run("returns false when no providers configured", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{}}
		_, _, ok := s.resolveCredentialProvider("user@example.com")
		assert.False(t, ok)
	})

	t.Run("local catch-all matches when no domain providers configured", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0},
		}}
		key, provider, ok := s.resolveCredentialProvider("user@anywhere.test")
		require.True(t, ok)
		assert.Equal(t, "local", key)
		assert.Equal(t, models.LocalProviderType, provider.Type)
	})

	t.Run("OIDC providers are ignored for credential flow", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"okta": {Type: models.OIDCProviderType, Order: 0, Domains: []string{"example.com"}},
		}}
		_, _, ok := s.resolveCredentialProvider("user@example.com")
		assert.False(t, ok, "should not match OIDC providers via credentials path")
	})

	t.Run("LDAP with matching domain wins over local fallback", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0},
			"ldap":  {Type: models.LDAPProviderType, Order: 1, Domains: []string{"corp.example"}},
		}}
		key, provider, ok := s.resolveCredentialProvider("user@corp.example")
		require.True(t, ok)
		assert.Equal(t, "ldap", key)
		assert.Equal(t, models.LDAPProviderType, provider.Type)
	})

	t.Run("local fallback matches when no domain provider claims the email", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0},
			"ldap":  {Type: models.LDAPProviderType, Order: 1, Domains: []string{"corp.example"}},
		}}
		key, provider, ok := s.resolveCredentialProvider("user@other.test")
		require.True(t, ok)
		assert.Equal(t, "local", key)
		assert.Equal(t, models.LocalProviderType, provider.Type)
	})

	t.Run("lowest Order wins on tie between two domain-matching providers", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"ldap-a": {Type: models.LDAPProviderType, Order: 5, Domains: []string{"corp.example"}},
			"ldap-b": {Type: models.LDAPProviderType, Order: 2, Domains: []string{"corp.example"}},
		}}
		key, _, ok := s.resolveCredentialProvider("user@corp.example")
		require.True(t, ok)
		assert.Equal(t, "ldap-b", key, "lower Order should win for deterministic selection")
	})

	t.Run("lowest Order wins on tie between two local fallbacks", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local-a": {Type: models.LocalProviderType, Order: 9},
			"local-b": {Type: models.LocalProviderType, Order: 3},
		}}
		key, _, ok := s.resolveCredentialProvider("user@anywhere.test")
		require.True(t, ok)
		assert.Equal(t, "local-b", key)
	})
}

func TestNormalizeExternalEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"jdoe@corp.example", "jdoe@corp.example"},
		{"  JDoe@Corp.Example  ", "jdoe@corp.example"},
		{"MIXED.Case+Tag@Example.ORG", "mixed.case+tag@example.org"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeExternalEmail(tc.in))
		})
	}
}
