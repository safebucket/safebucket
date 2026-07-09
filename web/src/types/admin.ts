export interface CreateUserPayload {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
}

export interface TimeSeriesPoint {
  timestamp: string;
  count: number;
}

export interface AdminStatsResponse {
  total_users: number;
  total_buckets: number;
  total_files: number;
  total_folders: number;
  total_storage: number;
  shared_files_per_hour: Array<TimeSeriesPoint>;
}

export interface IAdminBucket {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  creator: {
    id: string;
    first_name: string;
    last_name: string;
    email: string;
  };
  member_count: number;
  file_count: number;
  size: number;
}

export interface AdminAppSettings {
  profile: string;
  api_url: string;
  web_url: string;
  log_level: string;
  port: number;
  static_files_enabled: boolean;
  max_upload_size: number;
  trash_retention_days: number;
  allow_redirect_download: boolean;
  tls_enabled: boolean;
}

export interface AdminWorkerSettings {
  http_server: boolean;
  object_deletion: boolean;
  bucket_events: boolean;
  trash_cleanup: boolean;
  garbage_collector: boolean;
}

export interface AdminDatabaseSettings {
  type: string;
  host?: string;
  port?: number;
  name?: string;
  sslmode?: string;
  path?: string;
}

export interface AdminCacheSettings {
  type: string;
  hosts?: Array<string>;
  tls_enabled: boolean;
  tls_server_name?: string;
}

export interface AdminStorageSettings {
  type: string;
  bucket_name?: string;
  endpoint?: string;
  external_endpoint?: string;
  region?: string;
  project_id?: string;
  force_path_style?: boolean;
  use_tls?: boolean;
}

export interface AdminEventsSettings {
  type: string;
  queues?: Array<string>;
  host?: string;
  port?: string;
  project_id?: string;
  subscription_suffix?: string;
}

export interface AdminNotifierSettings {
  type: string;
  host?: string;
  port?: number;
  sender?: string;
  tls_mode?: string;
  skip_verify_tls?: boolean;
  directory?: string;
}

export interface AdminActivitySettings {
  type: string;
  endpoint?: string;
  directory?: string;
}

export interface AdminAuthProviderSettings {
  key: string;
  name?: string;
  type: string;
  domains?: Array<string>;
  mfa_required: boolean;
  sharing_allowed: boolean;
  issuer?: string;
  url?: string;
  base_dn?: string;
  user_filter?: string;
  start_tls?: boolean;
  tls_insecure_skip?: boolean;
  attribute_email?: string;
}

export interface AdminAuthSettings {
  providers: Array<AdminAuthProviderSettings>;
}

export interface AdminProfilingSettings {
  enabled: boolean;
  type?: string;
  server_address?: string;
  application_name?: string;
  upload_rate?: number;
}

export interface AdminTracingSettings {
  enabled: boolean;
  type?: string;
  endpoint?: string;
  service_name?: string;
  sampling_rate?: number;
}

export interface AdminObservabilitySettings {
  profiling: AdminProfilingSettings;
  tracing: AdminTracingSettings;
}

export interface AdminSecuritySettings {
  authenticated_requests_per_minute: number;
  unauthenticated_requests_per_minute: number;
  access_token_expiry: number;
  refresh_token_expiry: number;
  mfa_token_expiry: number;
  trusted_proxies?: Array<string>;
  allowed_origins?: Array<string>;
  cookie_secure_force: boolean;
}

export interface AdminSettingsResponse {
  platforms: number;
  app: AdminAppSettings;
  workers: AdminWorkerSettings;
  database: AdminDatabaseSettings;
  cache: AdminCacheSettings;
  storage: AdminStorageSettings;
  events: AdminEventsSettings;
  notifier: AdminNotifierSettings;
  activity: AdminActivitySettings;
  auth: AdminAuthSettings;
  observability: AdminObservabilitySettings;
  security: AdminSecuritySettings;
}
