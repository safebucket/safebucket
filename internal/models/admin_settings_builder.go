package models

import "sort"

func NewAdminSettingsResponse(
	cfg Configuration,
	platforms *int,
	coverage WorkerSettings,
) AdminSettingsResponse {
	return AdminSettingsResponse{
		Platforms:     platforms,
		App:           buildAppSettings(cfg.App),
		Workers:       coverage,
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
			settings.ForcePathStyle = boolPtr(storage.S3.ForcePathStyle)
			settings.UseTLS = boolPtr(storage.S3.UseTLS)
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
			settings.SkipVerifyTLS = boolPtr(notifier.SMTP.SkipVerifyTLS)
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
				settings.StartTLS = boolPtr(provider.LDAP.StartTLS)
				settings.TLSInsecureSkip = boolPtr(provider.LDAP.TLSInsecureSkip)
				settings.AttributeEmail = provider.LDAP.AttributeMap.Email
			}
		case LocalProviderType, ServiceAccountProviderType:
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

func boolPtr(b bool) *bool {
	return &b
}
