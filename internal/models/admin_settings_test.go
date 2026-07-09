package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullConfig returns a Configuration populated across every section with distinctive
// secret sentinels, so tests can assert those secrets never reach the response.
func fullConfig() Configuration {
	return Configuration{
		App: AppConfiguration{
			Profile:                          "default",
			AdminEmail:                       "admin@sentinel.io",
			AdminPassword:                    "SECRET-admin-pass",
			APIURL:                           "https://api.example.com",
			AllowedOrigins:                   []string{"https://app.example.com"},
			TokenSecret:                      "SECRET-token",
			MFAEncryptionKey:                 "SECRET-mfa-key-0123456789abcdef0",
			AccessTokenExpiry:                15,
			RefreshTokenExpiry:               600,
			MFATokenExpiry:                   5,
			LogLevel:                         "info",
			Port:                             8080,
			StaticFiles:                      StaticConfiguration{Enabled: true},
			TrustedProxies:                   []string{"10.0.0.0/8"},
			WebURL:                           "https://app.example.com",
			TrashRetentionDays:               30,
			MaxUploadSize:                    1024,
			AuthenticatedRequestsPerMinute:   200,
			UnauthenticatedRequestsPerMinute: 20,
			TLSCertFile:                      "/etc/tls/cert.pem",
			TLSKeyFile:                       "/etc/tls/key.pem",
			CookieSecureForce:                true,
			AllowRedirectDownload:            true,
		},
		Database: DatabaseConfiguration{
			Type: "postgres",
			Postgres: &PostgresDatabaseConfig{
				Host:     "db-host",
				Port:     5432,
				User:     "SECRET-db-user",
				Password: "SECRET-db-pass",
				Name:     "safebucket",
				SSLMode:  "disable",
			},
		},
		Auth: AuthConfiguration{
			Providers: map[string]ProviderConfiguration{
				"local": {Type: LocalProviderType},
				"okta": {
					Name: "Okta",
					Type: OIDCProviderType,
					OIDC: OIDCConfiguration{
						ClientID:     "SECRET-oidc-client-id",
						ClientSecret: "SECRET-oidc-secret",
						Issuer:       "https://okta.example.com",
					},
					Domains:     []string{"example.com"},
					MFARequired: true,
				},
				"corp": {
					Name: "Corp LDAP",
					Type: LDAPProviderType,
					LDAP: &LDAPConfiguration{
						URL:          "ldaps://ldap.example.com",
						BindDN:       "SECRET-bind-dn",
						BindPassword: "SECRET-bind-pass",
						BaseDN:       "dc=example,dc=com",
						UserFilter:   "(uid=%s)",
						AttributeMap: LDAPAttributeMap{Email: "mail"},
						StartTLS:     true,
					},
				},
			},
		},
		Cache: CacheConfiguration{
			Type: "redis",
			Redis: &RedisCacheConfiguration{
				Hosts:         []string{"redis-host:6379"},
				Password:      "SECRET-redis-pass",
				TLSEnabled:    true,
				TLSServerName: "redis.tls.example.com",
			},
		},
		Storage: StorageConfiguration{
			Type: "s3",
			S3: &S3Configuration{
				BucketName:     "safebucket-data",
				Endpoint:       "s3.example.com",
				AccessKey:      "SECRET-access-key",
				SecretKey:      "SECRET-secret-key",
				Region:         "eu-west-1",
				ForcePathStyle: true,
				UseTLS:         true,
			},
		},
		Events: EventsConfiguration{
			Type:      "jetstream",
			Queues:    map[string]QueueConfig{"notifications": {Name: "sb-notifications"}},
			Jetstream: &JetStreamEventsConfig{Host: "nats-host", Port: "4222"},
		},
		Notifier: NotifierConfiguration{
			Type: "smtp",
			SMTP: &MailerConfiguration{
				Host:          "smtp-host",
				Port:          587,
				Username:      "SECRET-smtp-user",
				Password:      "SECRET-smtp-pass",
				Sender:        "no-reply@example.com",
				TLSMode:       TLSModeStartTLS,
				SkipVerifyTLS: false,
			},
		},
		Activity: ActivityConfiguration{
			Type: "loki",
			Loki: &LokiConfiguration{Endpoint: "http://loki.example.com"},
		},
		Profiling: ProfilingConfiguration{
			Enabled: true,
			Type:    "pyroscope",
			Pyroscope: &PyroscopeConfiguration{
				ServerAddress:   "http://pyroscope.example.com",
				ApplicationName: "safebucket",
				UploadRate:      10,
			},
		},
		Tracing: TracingConfiguration{
			Enabled: true,
			Type:    "tempo",
			Tempo: &TempoConfiguration{
				Endpoint:     "http://tempo.example.com",
				ServiceName:  "safebucket",
				SamplingRate: 0.5,
			},
		},
	}
}

// coverageFixture mixes covered and uncovered components so tests assert both True and False.
func coverageFixture() WorkerCoverage {
	return WorkerCoverage{
		HTTPServer:       true,
		ObjectDeletion:   true,
		BucketEvents:     true,
		TrashCleanup:     true,
		GarbageCollector: false,
	}
}

func TestBuildAdminSettingsRedactsSecrets(t *testing.T) {
	settings := BuildAdminSettings(fullConfig(), 3, coverageFixture())

	raw, err := json.Marshal(settings)
	require.NoError(t, err)
	payload := string(raw)

	secrets := []string{
		"SECRET-admin-pass",
		"SECRET-token",
		"SECRET-mfa-key-0123456789abcdef0",
		"SECRET-db-user",
		"SECRET-db-pass",
		"SECRET-oidc-client-id",
		"SECRET-oidc-secret",
		"SECRET-bind-dn",
		"SECRET-bind-pass",
		"SECRET-redis-pass",
		"SECRET-access-key",
		"SECRET-secret-key",
		"SECRET-smtp-user",
		"SECRET-smtp-pass",
		"admin@sentinel.io",
	}
	for _, secret := range secrets {
		assert.NotContains(t, payload, secret, "response must not expose %q", secret)
	}
}

func TestBuildAdminSettingsExposesNonSecrets(t *testing.T) {
	settings := BuildAdminSettings(fullConfig(), 3, coverageFixture())

	assert.Equal(t, 3, settings.Platforms)
	assert.Equal(t, "default", settings.App.Profile)
	assert.True(t, settings.App.TLSEnabled)
	assert.True(t, settings.App.StaticFilesEnabled)

	assert.True(t, settings.Workers.ObjectDeletion)
	assert.False(t, settings.Workers.GarbageCollector)
	assert.True(t, settings.Workers.HTTPServer)

	assert.Equal(t, "postgres", settings.Database.Type)
	assert.Equal(t, "db-host", settings.Database.Host)
	assert.Equal(t, "safebucket", settings.Database.Name)

	assert.Equal(t, "redis", settings.Cache.Type)
	assert.Equal(t, []string{"redis-host:6379"}, settings.Cache.Hosts)
	assert.True(t, settings.Cache.TLSEnabled)

	assert.Equal(t, "s3", settings.Storage.Type)
	assert.Equal(t, "safebucket-data", settings.Storage.BucketName)
	assert.Equal(t, "s3.example.com", settings.Storage.Endpoint)
	require.NotNil(t, settings.Storage.ForcePathStyle)
	assert.True(t, *settings.Storage.ForcePathStyle)

	assert.Equal(t, "jetstream", settings.Events.Type)
	assert.Equal(t, "nats-host", settings.Events.Host)
	assert.Equal(t, []string{"sb-notifications"}, settings.Events.Queues)

	assert.Equal(t, "smtp", settings.Notifier.Type)
	assert.Equal(t, "no-reply@example.com", settings.Notifier.Sender)

	assert.Equal(t, "loki", settings.Activity.Type)
	assert.Equal(t, "http://loki.example.com", settings.Activity.Endpoint)

	assert.True(t, settings.Observability.Profiling.Enabled)
	assert.Equal(t, "http://tempo.example.com", settings.Observability.Tracing.Endpoint)

	assert.Equal(t, 200, settings.Security.AuthenticatedRequestsPerMinute)
	assert.Equal(t, []string{"https://app.example.com"}, settings.Security.AllowedOrigins)
}

func TestBuildAdminSettingsAuthProviders(t *testing.T) {
	settings := BuildAdminSettings(fullConfig(), 3, coverageFixture())

	require.Len(t, settings.Auth.Providers, 3)

	byKey := map[string]AuthProviderSettings{}
	for _, provider := range settings.Auth.Providers {
		byKey[provider.Key] = provider
	}

	okta := byKey["okta"]
	assert.Equal(t, string(OIDCProviderType), okta.Type)
	assert.Equal(t, "https://okta.example.com", okta.Issuer)
	assert.True(t, okta.MFARequired)
	assert.Empty(t, okta.URL)

	corp := byKey["corp"]
	assert.Equal(t, string(LDAPProviderType), corp.Type)
	assert.Equal(t, "ldaps://ldap.example.com", corp.URL)
	assert.Equal(t, "dc=example,dc=com", corp.BaseDN)
	require.NotNil(t, corp.StartTLS)
	assert.True(t, *corp.StartTLS)
	assert.Empty(t, corp.Issuer)

	local := byKey["local"]
	assert.Equal(t, string(LocalProviderType), local.Type)
}
