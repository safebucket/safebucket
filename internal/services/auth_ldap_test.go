package services

import (
	"errors"
	"testing"

	"github.com/safebucket/safebucket/internal/auth/ldap"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

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
		assert.False(t, ok)
	})

	t.Run("LDAP providers are ignored for credential flow", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"ldap-domain":   {Type: models.LDAPProviderType, Order: 0, Domains: []string{"corp.example"}},
			"ldap-catchall": {Type: models.LDAPProviderType, Order: 1},
		}}
		_, _, ok := s.resolveCredentialProvider("user@corp.example")
		assert.False(t, ok)
	})

	t.Run("local fallback matches even when an LDAP domain provider claims the email", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0},
			"ldap":  {Type: models.LDAPProviderType, Order: 1, Domains: []string{"corp.example"}},
		}}
		key, provider, ok := s.resolveCredentialProvider("user@corp.example")
		require.True(t, ok)
		assert.Equal(t, "local", key)
		assert.Equal(t, models.LocalProviderType, provider.Type)
	})

	t.Run("local domain provider rejects an email outside its domains", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0, Domains: []string{"corp.example"}},
		}}
		_, _, ok := s.resolveCredentialProvider("user@anywhere.test")
		assert.False(t, ok)
	})

	t.Run("local domain provider matches an email within its domains", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"local": {Type: models.LocalProviderType, Order: 0, Domains: []string{"corp.example"}},
		}}
		key, _, ok := s.resolveCredentialProvider("user@corp.example")
		require.True(t, ok)
		assert.Equal(t, "local", key)
	})
}

func TestLDAPLogin(t *testing.T) {
	body := models.AuthLoginBody{Email: "user@corp.example", Password: "secret"}

	assertAPIErrorCode := func(t *testing.T, err error, code int) {
		t.Helper()
		require.Error(t, err)
		var apiErr *apierrors.APIError
		require.True(t, errors.As(err, &apiErr), "expected an APIError")
		assert.Equal(t, code, apiErr.Code)
	}

	t.Run("returns 404 for unknown provider", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{}}
		_, err := s.LDAPLogin(false, zap.NewNop(), models.UserClaims{}, uuid.UUIDs{}, "missing", body)
		assertAPIErrorCode(t, err, 404)
	})

	t.Run("returns 404 when the provider is not LDAP", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"okta":  {Type: models.OIDCProviderType},
			"local": {Type: models.LocalProviderType},
		}}

		_, err := s.LDAPLogin(false, zap.NewNop(), models.UserClaims{}, uuid.UUIDs{}, "okta", body)
		assertAPIErrorCode(t, err, 404)

		_, err = s.LDAPLogin(false, zap.NewNop(), models.UserClaims{}, uuid.UUIDs{}, "local", body)
		assertAPIErrorCode(t, err, 404)
	})

	t.Run("returns internal error when LDAP client config is missing", func(t *testing.T) {
		s := AuthService{Providers: configuration.Providers{
			"ldap": {Type: models.LDAPProviderType},
		}}
		_, err := s.LDAPLogin(false, zap.NewNop(), models.UserClaims{}, uuid.UUIDs{}, "ldap", body)
		require.ErrorIs(t, err, apierrors.ErrInternalServer)
	})
}

func TestMapLDAPAuthError(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode int
		wantMsg  string
	}{
		{"invalid credentials stays 401", ldap.ErrInvalidCredentials, 401, apierrors.CodeInvalidCredentials},
		{"unknown user looks like bad password", ldap.ErrUserNotFound, 401, apierrors.CodeInvalidCredentials},
		{"directory unavailable is 503", ldap.ErrDirectoryUnavailable, 503, apierrors.CodeAuthProviderUnavailable},
		{"service bind failure is 503", ldap.ErrServiceBindFailed, 503, apierrors.CodeAuthProviderUnavailable},
		{"missing email is a server fault", ldap.ErrMissingEmail, 503, apierrors.CodeAuthProviderUnavailable},
		{
			"unexpected error surfaces as 503",
			errors.New("matched 3 entries"),
			503,
			apierrors.CodeAuthProviderUnavailable,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := mapLDAPAuthError(zap.NewNop(), "ldap", tc.err, apierrors.CodeInvalidCredentials)
			var apiErr *apierrors.APIError
			require.True(t, errors.As(err, &apiErr), "expected an APIError")
			assert.Equal(t, tc.wantCode, apiErr.Code)
			assert.Equal(t, tc.wantMsg, apiErr.Error())
		})
	}
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
