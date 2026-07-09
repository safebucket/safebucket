package models

import "sort"

// AdminSettingsResponse is a read-only, secret-free view of the loaded configuration,
// returned by GET /api/v1/admin/settings. It is built by BuildAdminSettings, which
// whitelists only non-sensitive fields; the raw Configuration is never serialized.
type AdminSettingsResponse struct {
	Platforms     int                   `json:"platforms"`
	App           AppSettings           `json:"app"`
	Workers       WorkerSettings        `json:"workers"`
	Database      DatabaseSettings      `json:"database"`
	Cache         CacheSettings         `json:"cache"`
	Storage       StorageSettings       `json:"storage"`
	Events        EventsSettings        `json:"events"`
	Notifier      NotifierSettings      `json:"notifier"`
	Activity      ActivitySettings      `json:"activity"`
	Auth          AuthSettings          `json:"auth"`
	Observability ObservabilitySettings `json:"observability"`
	Security      SecuritySettings      `json:"security"`
}

type AppSettings struct {
	Profile               string `json:"profile"`
	APIURL                string `json:"api_url"`
	WebURL                string `json:"web_url"`
	LogLevel              string `json:"log_level"`
	Port                  int    `json:"port"`
	StaticFilesEnabled    bool   `json:"static_files_enabled"`
	MaxUploadSize         int64  `json:"max_upload_size"`
	TrashRetentionDays    int    `json:"trash_retention_days"`
	AllowRedirectDownload bool   `json:"allow_redirect_download"`
	TLSEnabled            bool   `json:"tls_enabled"`
}

// WorkerSettings reports, per component, whether it is covered by a live instance in the fleet
// (Enabled) or not running anywhere (Disabled).
type WorkerSettings struct {
	HTTPServer       bool `json:"http_server"`
	ObjectDeletion   bool `json:"object_deletion"`
	BucketEvents     bool `json:"bucket_events"`
	TrashCleanup     bool `json:"trash_cleanup"`
	GarbageCollector bool `json:"garbage_collector"`
}

// WorkerCoverage carries the coverage booleans the admin service resolves from the cache.
type WorkerCoverage struct {
	HTTPServer       bool
	ObjectDeletion   bool
	BucketEvents     bool
	TrashCleanup     bool
	GarbageCollector bool
}

type DatabaseSettings struct {
	Type    string `json:"type"`
	Host    string `json:"host,omitempty"`
	Port    int32  `json:"port,omitempty"`
	Name    string `json:"name,omitempty"`
	SSLMode string `json:"sslmode,omitempty"`
	Path    string `json:"path,omitempty"`
}

type CacheSettings struct {
	Type          string   `json:"type"`
	Hosts         []string `json:"hosts,omitempty"`
	TLSEnabled    bool     `json:"tls_enabled"`
	TLSServerName string   `json:"tls_server_name,omitempty"`
}

type StorageSettings struct {
	Type             string `json:"type"`
	BucketName       string `json:"bucket_name,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	ExternalEndpoint string `json:"external_endpoint,omitempty"`
	Region           string `json:"region,omitempty"`
	ProjectID        string `json:"project_id,omitempty"`
	ForcePathStyle   *bool  `json:"force_path_style,omitempty"`
	UseTLS           *bool  `json:"use_tls,omitempty"`
}

type EventsSettings struct {
	Type               string   `json:"type"`
	Queues             []string `json:"queues,omitempty"`
	Host               string   `json:"host,omitempty"`
	Port               string   `json:"port,omitempty"`
	ProjectID          string   `json:"project_id,omitempty"`
	SubscriptionSuffix string   `json:"subscription_suffix,omitempty"`
}

type NotifierSettings struct {
	Type          string `json:"type"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	Sender        string `json:"sender,omitempty"`
	TLSMode       string `json:"tls_mode,omitempty"`
	SkipVerifyTLS *bool  `json:"skip_verify_tls,omitempty"`
	Directory     string `json:"directory,omitempty"`
}

type ActivitySettings struct {
	Type      string `json:"type"`
	Endpoint  string `json:"endpoint,omitempty"`
	Directory string `json:"directory,omitempty"`
}

type AuthSettings struct {
	Providers []AuthProviderSettings `json:"providers"`
}

type AuthProviderSettings struct {
	Key             string   `json:"key"`
	Name            string   `json:"name,omitempty"`
	Type            string   `json:"type"`
	Domains         []string `json:"domains,omitempty"`
	MFARequired     bool     `json:"mfa_required"`
	SharingAllowed  bool     `json:"sharing_allowed"`
	Issuer          string   `json:"issuer,omitempty"`
	URL             string   `json:"url,omitempty"`
	BaseDN          string   `json:"base_dn,omitempty"`
	UserFilter      string   `json:"user_filter,omitempty"`
	StartTLS        *bool    `json:"start_tls,omitempty"`
	TLSInsecureSkip *bool    `json:"tls_insecure_skip,omitempty"`
	AttributeEmail  string   `json:"attribute_email,omitempty"`
}

type ObservabilitySettings struct {
	Profiling ProfilingSettings `json:"profiling"`
	Tracing   TracingSettings   `json:"tracing"`
}

type ProfilingSettings struct {
	Enabled         bool   `json:"enabled"`
	Type            string `json:"type,omitempty"`
	ServerAddress   string `json:"server_address,omitempty"`
	ApplicationName string `json:"application_name,omitempty"`
	UploadRate      int    `json:"upload_rate,omitempty"`
}

type TracingSettings struct {
	Enabled      bool    `json:"enabled"`
	Type         string  `json:"type,omitempty"`
	Endpoint     string  `json:"endpoint,omitempty"`
	ServiceName  string  `json:"service_name,omitempty"`
	SamplingRate float64 `json:"sampling_rate,omitempty"`
}

type SecuritySettings struct {
	AuthenticatedRequestsPerMinute   int      `json:"authenticated_requests_per_minute"`
	UnauthenticatedRequestsPerMinute int      `json:"unauthenticated_requests_per_minute"`
	AccessTokenExpiry                int      `json:"access_token_expiry"`
	RefreshTokenExpiry               int      `json:"refresh_token_expiry"`
	MFATokenExpiry                   int      `json:"mfa_token_expiry"`
	TrustedProxies                   []string `json:"trusted_proxies,omitempty"`
	AllowedOrigins                   []string `json:"allowed_origins,omitempty"`
	CookieSecureForce                bool     `json:"cookie_secure_force"`
}

// BuildAdminSettings assembles the secret-free settings view from the loaded
// configuration, the resolved runtime profile, and the number of active app
// instances (platforms). Every secret (token secret, passwords, access keys,
// client secrets, bind passwords) is intentionally dropped, along with
// credential-adjacent identifiers (admin email, DB user, SMTP username, OIDC
// client id, LDAP bind DN).
func BuildAdminSettings(
	cfg Configuration,
	platforms int,
	coverage WorkerCoverage,
) AdminSettingsResponse {
	return AdminSettingsResponse{
		Platforms:     platforms,
		App:           buildAppSettings(cfg.App),
		Workers:       buildWorkerSettings(coverage),
		Database:      buildDatabaseSettings(cfg.Database),
		Cache:         buildCacheSettings(cfg.Cache),
		Storage:       buildStorageSettings(cfg.Storage),
		Events:        buildEventsSettings(cfg.Events),
		Notifier:      buildNotifierSettings(cfg.Notifier),
		Activity:      buildActivitySettings(cfg.Activity),
		Auth:          buildAuthSettings(cfg.Auth),
		Observability: buildObservabilitySettings(cfg),
		Security:      buildSecuritySettings(cfg.App),
	}
}

func buildAppSettings(app AppConfiguration) AppSettings {
	return AppSettings{
		Profile:               app.Profile,
		APIURL:                app.APIURL,
		WebURL:                app.WebURL,
		LogLevel:              app.LogLevel,
		Port:                  app.Port,
		StaticFilesEnabled:    app.StaticFiles.Enabled,
		MaxUploadSize:         app.MaxUploadSize,
		TrashRetentionDays:    app.TrashRetentionDays,
		AllowRedirectDownload: app.AllowRedirectDownload,
		TLSEnabled:            app.TLSCertFile != "" && app.TLSKeyFile != "",
	}
}

// buildWorkerSettings relies on WorkerSettings and WorkerCoverage keeping identical field order.
func buildWorkerSettings(coverage WorkerCoverage) WorkerSettings {
	return WorkerSettings(coverage)
}

func buildDatabaseSettings(db DatabaseConfiguration) DatabaseSettings {
	settings := DatabaseSettings{Type: db.Type}

	switch db.Type {
	case "postgres":
		if db.Postgres != nil {
			settings.Host = db.Postgres.Host
			settings.Port = db.Postgres.Port
			settings.Name = db.Postgres.Name
			settings.SSLMode = db.Postgres.SSLMode
		}
	case "sqlite":
		if db.SQLite != nil {
			settings.Path = db.SQLite.Path
		}
	}

	return settings
}

func buildCacheSettings(cache CacheConfiguration) CacheSettings {
	settings := CacheSettings{Type: cache.Type}

	switch cache.Type {
	case "redis":
		if cache.Redis != nil {
			settings.Hosts = cache.Redis.Hosts
			settings.TLSEnabled = cache.Redis.TLSEnabled
			settings.TLSServerName = cache.Redis.TLSServerName
		}
	case "valkey":
		if cache.Valkey != nil {
			settings.Hosts = cache.Valkey.Hosts
			settings.TLSEnabled = cache.Valkey.TLSEnabled
			settings.TLSServerName = cache.Valkey.TLSServerName
		}
	}

	return settings
}

func buildStorageSettings(storage StorageConfiguration) StorageSettings {
	settings := StorageSettings{Type: storage.Type}

	switch storage.Type {
	case "minio":
		if storage.Minio != nil {
			settings.BucketName = storage.Minio.BucketName
			settings.Endpoint = storage.Minio.Endpoint
			settings.ExternalEndpoint = storage.Minio.ExternalEndpoint
			settings.Region = storage.Minio.Region
		}
	case "rustfs":
		if storage.RustFS != nil {
			settings.BucketName = storage.RustFS.BucketName
			settings.Endpoint = storage.RustFS.Endpoint
			settings.ExternalEndpoint = storage.RustFS.ExternalEndpoint
			settings.Region = storage.RustFS.Region
		}
	case "s3":
		if storage.S3 != nil {
			settings.BucketName = storage.S3.BucketName
			settings.Endpoint = storage.S3.Endpoint
			settings.ExternalEndpoint = storage.S3.ExternalEndpoint
			settings.Region = storage.S3.Region
			settings.ForcePathStyle = ptrOf(storage.S3.ForcePathStyle)
			settings.UseTLS = ptrOf(storage.S3.UseTLS)
		}
	case "gcp":
		if storage.CloudStorage != nil {
			settings.BucketName = storage.CloudStorage.BucketName
			settings.ExternalEndpoint = storage.CloudStorage.ExternalEndpoint
			settings.ProjectID = storage.CloudStorage.ProjectID
		}
	case "aws":
		if storage.AWS != nil {
			settings.BucketName = storage.AWS.BucketName
			settings.ExternalEndpoint = storage.AWS.ExternalEndpoint
		}
	}

	return settings
}

func buildEventsSettings(events EventsConfiguration) EventsSettings {
	settings := EventsSettings{Type: events.Type}

	if len(events.Queues) > 0 {
		queues := make([]string, 0, len(events.Queues))
		for _, queue := range events.Queues {
			queues = append(queues, queue.Name)
		}
		sort.Strings(queues)
		settings.Queues = queues
	}

	switch events.Type {
	case "jetstream":
		if events.Jetstream != nil {
			settings.Host = events.Jetstream.Host
			settings.Port = events.Jetstream.Port
		}
	case "gcp":
		if events.PubSub != nil {
			settings.ProjectID = events.PubSub.ProjectID
			settings.SubscriptionSuffix = events.PubSub.SubscriptionSuffix
		}
	}

	return settings
}

func buildNotifierSettings(notifier NotifierConfiguration) NotifierSettings {
	settings := NotifierSettings{Type: notifier.Type}

	switch notifier.Type {
	case "smtp":
		if notifier.SMTP != nil {
			settings.Host = notifier.SMTP.Host
			settings.Port = notifier.SMTP.Port
			settings.Sender = notifier.SMTP.Sender
			settings.TLSMode = string(notifier.SMTP.TLSMode)
			settings.SkipVerifyTLS = ptrOf(notifier.SMTP.SkipVerifyTLS)
		}
	case "filesystem":
		if notifier.Filesystem != nil {
			settings.Directory = notifier.Filesystem.Directory
		}
	}

	return settings
}

func buildActivitySettings(activity ActivityConfiguration) ActivitySettings {
	settings := ActivitySettings{Type: activity.Type}

	switch activity.Type {
	case "loki":
		if activity.Loki != nil {
			settings.Endpoint = activity.Loki.Endpoint
		}
	case "filesystem":
		if activity.Filesystem != nil {
			settings.Directory = activity.Filesystem.Directory
		}
	}

	return settings
}

func buildAuthSettings(auth AuthConfiguration) AuthSettings {
	keys := make([]string, 0, len(auth.Providers))
	for key := range auth.Providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	providers := make([]AuthProviderSettings, 0, len(keys))
	for _, key := range keys {
		provider := auth.Providers[key]
		settings := AuthProviderSettings{
			Key:            key,
			Name:           provider.Name,
			Type:           string(provider.Type),
			Domains:        provider.Domains,
			MFARequired:    provider.MFARequired,
			SharingAllowed: provider.SharingConfiguration.Allowed,
		}

		switch provider.Type {
		case OIDCProviderType:
			settings.Issuer = provider.OIDC.Issuer
		case LDAPProviderType:
			if provider.LDAP != nil {
				settings.URL = provider.LDAP.URL
				settings.BaseDN = provider.LDAP.BaseDN
				settings.UserFilter = provider.LDAP.UserFilter
				settings.StartTLS = ptrOf(provider.LDAP.StartTLS)
				settings.TLSInsecureSkip = ptrOf(provider.LDAP.TLSInsecureSkip)
				settings.AttributeEmail = provider.LDAP.AttributeMap.Email
			}
		case LocalProviderType:
		}

		providers = append(providers, settings)
	}

	return AuthSettings{Providers: providers}
}

func buildObservabilitySettings(cfg Configuration) ObservabilitySettings {
	profiling := ProfilingSettings{
		Enabled: cfg.Profiling.Enabled,
		Type:    cfg.Profiling.Type,
	}
	if cfg.Profiling.Pyroscope != nil {
		profiling.ServerAddress = cfg.Profiling.Pyroscope.ServerAddress
		profiling.ApplicationName = cfg.Profiling.Pyroscope.ApplicationName
		profiling.UploadRate = cfg.Profiling.Pyroscope.UploadRate
	}

	tracing := TracingSettings{
		Enabled: cfg.Tracing.Enabled,
		Type:    cfg.Tracing.Type,
	}
	if cfg.Tracing.Tempo != nil {
		tracing.Endpoint = cfg.Tracing.Tempo.Endpoint
		tracing.ServiceName = cfg.Tracing.Tempo.ServiceName
		tracing.SamplingRate = cfg.Tracing.Tempo.SamplingRate
	}

	return ObservabilitySettings{Profiling: profiling, Tracing: tracing}
}

func buildSecuritySettings(app AppConfiguration) SecuritySettings {
	return SecuritySettings{
		AuthenticatedRequestsPerMinute:   app.AuthenticatedRequestsPerMinute,
		UnauthenticatedRequestsPerMinute: app.UnauthenticatedRequestsPerMinute,
		AccessTokenExpiry:                app.AccessTokenExpiry,
		RefreshTokenExpiry:               app.RefreshTokenExpiry,
		MFATokenExpiry:                   app.MFATokenExpiry,
		TrustedProxies:                   app.TrustedProxies,
		AllowedOrigins:                   app.AllowedOrigins,
		CookieSecureForce:                app.CookieSecureForce,
	}
}

func ptrOf[T any](v T) *T {
	return &v
}
