package models

type AdminSettingsResponse struct {
	Platforms     *int                  `json:"platforms"`
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
	MaxFileVersions       int    `json:"max_file_versions"`
	AllowRedirectDownload bool   `json:"allow_redirect_download"`
	TLSEnabled            bool   `json:"tls_enabled"`
}

type CoverageStatus string

const (
	CoverageCovered       CoverageStatus = "covered"
	CoverageNotCovered    CoverageStatus = "not_covered"
	CoverageNotApplicable CoverageStatus = "not_applicable"
	CoverageUnknown       CoverageStatus = "unknown"
)

type WorkerSettings struct {
	HTTPServer       CoverageStatus `json:"http_server"`
	ObjectDeletion   CoverageStatus `json:"object_deletion"`
	BucketEvents     CoverageStatus `json:"bucket_events"`
	TrashCleanup     CoverageStatus `json:"trash_cleanup"`
	GarbageCollector CoverageStatus `json:"garbage_collector"`
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
