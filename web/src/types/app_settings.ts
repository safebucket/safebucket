export interface IAdminAppSettings {
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

export type CoverageStatus =
  "covered" | "not_covered" | "not_applicable" | "unknown";

export interface IAdminWorkerSettings {
  http_server: CoverageStatus;
  object_deletion: CoverageStatus;
  bucket_events: CoverageStatus;
  trash_cleanup: CoverageStatus;
  garbage_collector: CoverageStatus;
}

export interface IAdminDatabaseSettings {
  type: string;
  host?: string;
  port?: number;
  name?: string;
  sslmode?: string;
  path?: string;
}

export interface IAdminCacheSettings {
  type: string;
  hosts?: Array<string>;
  tls_enabled: boolean;
  tls_server_name?: string;
}

export interface IAdminStorageSettings {
  type: string;
  bucket_name?: string;
  endpoint?: string;
  external_endpoint?: string;
  region?: string;
  project_id?: string;
  force_path_style?: boolean;
  use_tls?: boolean;
}

export interface IAdminEventsSettings {
  type: string;
  queues?: Array<string>;
  host?: string;
  port?: string;
  project_id?: string;
  subscription_suffix?: string;
}

export interface IAdminNotifierSettings {
  type: string;
  host?: string;
  port?: number;
  sender?: string;
  tls_mode?: string;
  skip_verify_tls?: boolean;
  directory?: string;
}

export interface IAdminActivitySettings {
  type: string;
  endpoint?: string;
  directory?: string;
}

export interface IAdminAuthProviderSettings {
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

export interface IAdminAuthSettings {
  providers: Array<IAdminAuthProviderSettings>;
}

export interface IAdminProfilingSettings {
  enabled: boolean;
  type?: string;
  server_address?: string;
  application_name?: string;
  upload_rate?: number;
}

export interface IAdminTracingSettings {
  enabled: boolean;
  type?: string;
  endpoint?: string;
  service_name?: string;
  sampling_rate?: number;
}

export interface IAdminObservabilitySettings {
  profiling: IAdminProfilingSettings;
  tracing: IAdminTracingSettings;
}

export interface IAdminSecuritySettings {
  authenticated_requests_per_minute: number;
  unauthenticated_requests_per_minute: number;
  access_token_expiry: number;
  refresh_token_expiry: number;
  mfa_token_expiry: number;
  trusted_proxies?: Array<string>;
  allowed_origins?: Array<string>;
  cookie_secure_force: boolean;
}

export interface IAdminSettingsResponse {
  platforms: number | null;
  app: IAdminAppSettings;
  workers: IAdminWorkerSettings;
  database: IAdminDatabaseSettings;
  cache: IAdminCacheSettings;
  storage: IAdminStorageSettings;
  events: IAdminEventsSettings;
  notifier: IAdminNotifierSettings;
  activity: IAdminActivitySettings;
  auth: IAdminAuthSettings;
  observability: IAdminObservabilitySettings;
  security: IAdminSecuritySettings;
}
