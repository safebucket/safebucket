import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import {
  ArrowLeft,
  Boxes,
  Cpu,
  Database,
  Gauge,
  HardDrive,
  Lock,
  Mail,
  Radio,
  ScrollText,
  ShieldCheck,
  SlidersHorizontal,
} from "lucide-react";
import { useAdminSettings } from "./hooks/useAdminSettings";
import { SettingsSection } from "./components/SettingsSection";
import { SettingRow } from "./components/SettingRow";
import { SettingsNav } from "./components/SettingsNav";
import { SettingsError } from "./components/SettingsError";
import { AuthProviderBlock } from "./components/AuthProviderBlock";
import { BoolValue } from "./components/BoolValue";
import { BytesValue } from "./components/BytesValue";
import { CopyableValue } from "./components/CopyableValue";
import { CoverageValue } from "./components/CoverageValue";
import { DurationValue } from "./components/DurationValue";
import { ListValue } from "./components/ListValue";
import { RateValue } from "./components/RateValue";
import { TextValue } from "./components/TextValue";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export function AdminSettingsDetails() {
  const { t } = useTranslation();
  const { data: settings, isLoading, isError, refetch } = useAdminSettings();

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="container mx-auto max-w-5xl p-6">
        <div className="mb-6">
          <Button asChild variant="ghost" size="sm" className="-ml-2 mb-2">
            <Link to="/admin/settings">
              <ArrowLeft className="h-4 w-4" />
              {t("admin.settings.back")}
            </Link>
          </Button>
          <h1 className="text-2xl font-semibold">
            {t("admin.settings.details_title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.details_description")}
          </p>
        </div>

        {isError ? (
          <SettingsError onRetry={() => void refetch()} />
        ) : isLoading || !settings ? (
          <div className="space-y-6">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton key={index} className="h-40 w-full" />
            ))}
          </div>
        ) : (
          <div className="flex gap-8">
            <SettingsNav />
            <div className="min-w-0 flex-1 space-y-6">
              <SettingsSection
                id="app"
                title={t("admin.settings.sections.app")}
                icon={SlidersHorizontal}
              >
                <SettingRow label={t("admin.settings.fields.profile")}>
                  <Badge variant="outline">{settings.app.profile}</Badge>
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.api_url")}>
                  <CopyableValue value={settings.app.api_url} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.web_url")}>
                  <CopyableValue value={settings.app.web_url} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.log_level")}>
                  <Badge variant="outline">{settings.app.log_level}</Badge>
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.port")}>
                  <TextValue value={settings.app.port} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.static_files")}>
                  <BoolValue value={settings.app.static_files_enabled} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.max_upload_size")}>
                  <BytesValue value={settings.app.max_upload_size} />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.trash_retention_days")}
                >
                  <TextValue value={settings.app.trash_retention_days} />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.allow_redirect_download")}
                >
                  <BoolValue value={settings.app.allow_redirect_download} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.tls")}>
                  <BoolValue value={settings.app.tls_enabled} />
                </SettingRow>
              </SettingsSection>

              <SettingsSection
                id="workers"
                title={t("admin.settings.sections.workers")}
                description={t("admin.settings.sections.workers_description")}
                icon={Cpu}
              >
                <SettingRow label={t("admin.settings.fields.platforms")}>
                  <TextValue value={settings.platforms} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.http_server")}>
                  <CoverageValue covered={settings.workers.http_server} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.object_deletion")}>
                  <CoverageValue covered={settings.workers.object_deletion} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.bucket_events")}>
                  <CoverageValue covered={settings.workers.bucket_events} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.trash_cleanup")}>
                  <CoverageValue covered={settings.workers.trash_cleanup} />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.garbage_collector")}
                >
                  <CoverageValue covered={settings.workers.garbage_collector} />
                </SettingRow>
              </SettingsSection>

              <SettingsSection
                id="database"
                title={t("admin.settings.sections.database")}
                icon={Database}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.database.type}</Badge>
                </SettingRow>
                {settings.database.host && (
                  <SettingRow label={t("admin.settings.fields.host")}>
                    <TextValue value={settings.database.host} />
                  </SettingRow>
                )}
                {settings.database.port !== undefined && (
                  <SettingRow label={t("admin.settings.fields.port")}>
                    <TextValue value={settings.database.port} />
                  </SettingRow>
                )}
                {settings.database.name && (
                  <SettingRow label={t("admin.settings.fields.name")}>
                    <TextValue value={settings.database.name} />
                  </SettingRow>
                )}
                {settings.database.sslmode && (
                  <SettingRow label={t("admin.settings.fields.sslmode")}>
                    <TextValue value={settings.database.sslmode} />
                  </SettingRow>
                )}
                {settings.database.path && (
                  <SettingRow label={t("admin.settings.fields.path")}>
                    <TextValue value={settings.database.path} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="cache"
                title={t("admin.settings.sections.cache")}
                icon={Boxes}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.cache.type}</Badge>
                </SettingRow>
                {settings.cache.hosts && (
                  <SettingRow label={t("admin.settings.fields.hosts")}>
                    <ListValue values={settings.cache.hosts} />
                  </SettingRow>
                )}
                <SettingRow label={t("admin.settings.fields.tls")}>
                  <BoolValue value={settings.cache.tls_enabled} />
                </SettingRow>
                {settings.cache.tls_server_name && (
                  <SettingRow
                    label={t("admin.settings.fields.tls_server_name")}
                  >
                    <TextValue value={settings.cache.tls_server_name} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="storage"
                title={t("admin.settings.sections.storage")}
                icon={HardDrive}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.storage.type}</Badge>
                </SettingRow>
                {settings.storage.bucket_name && (
                  <SettingRow label={t("admin.settings.fields.bucket_name")}>
                    <TextValue value={settings.storage.bucket_name} />
                  </SettingRow>
                )}
                {settings.storage.endpoint && (
                  <SettingRow label={t("admin.settings.fields.endpoint")}>
                    <CopyableValue value={settings.storage.endpoint} />
                  </SettingRow>
                )}
                {settings.storage.external_endpoint && (
                  <SettingRow
                    label={t("admin.settings.fields.external_endpoint")}
                  >
                    <CopyableValue value={settings.storage.external_endpoint} />
                  </SettingRow>
                )}
                {settings.storage.region && (
                  <SettingRow label={t("admin.settings.fields.region")}>
                    <TextValue value={settings.storage.region} />
                  </SettingRow>
                )}
                {settings.storage.project_id && (
                  <SettingRow label={t("admin.settings.fields.project_id")}>
                    <TextValue value={settings.storage.project_id} />
                  </SettingRow>
                )}
                {settings.storage.force_path_style !== undefined && (
                  <SettingRow
                    label={t("admin.settings.fields.force_path_style")}
                  >
                    <BoolValue value={settings.storage.force_path_style} />
                  </SettingRow>
                )}
                {settings.storage.use_tls !== undefined && (
                  <SettingRow label={t("admin.settings.fields.use_tls")}>
                    <BoolValue value={settings.storage.use_tls} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="events"
                title={t("admin.settings.sections.events")}
                icon={Radio}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.events.type}</Badge>
                </SettingRow>
                {settings.events.queues && (
                  <SettingRow label={t("admin.settings.fields.queues")}>
                    <ListValue values={settings.events.queues} />
                  </SettingRow>
                )}
                {settings.events.host && (
                  <SettingRow label={t("admin.settings.fields.host")}>
                    <CopyableValue value={settings.events.host} />
                  </SettingRow>
                )}
                {settings.events.port && (
                  <SettingRow label={t("admin.settings.fields.port")}>
                    <TextValue value={settings.events.port} />
                  </SettingRow>
                )}
                {settings.events.project_id && (
                  <SettingRow label={t("admin.settings.fields.project_id")}>
                    <TextValue value={settings.events.project_id} />
                  </SettingRow>
                )}
                {settings.events.subscription_suffix && (
                  <SettingRow
                    label={t("admin.settings.fields.subscription_suffix")}
                  >
                    <TextValue value={settings.events.subscription_suffix} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="notifier"
                title={t("admin.settings.sections.notifier")}
                icon={Mail}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.notifier.type}</Badge>
                </SettingRow>
                {settings.notifier.host && (
                  <SettingRow label={t("admin.settings.fields.host")}>
                    <CopyableValue value={settings.notifier.host} />
                  </SettingRow>
                )}
                {settings.notifier.port !== undefined && (
                  <SettingRow label={t("admin.settings.fields.port")}>
                    <TextValue value={settings.notifier.port} />
                  </SettingRow>
                )}
                {settings.notifier.sender && (
                  <SettingRow label={t("admin.settings.fields.sender")}>
                    <TextValue value={settings.notifier.sender} />
                  </SettingRow>
                )}
                {settings.notifier.tls_mode && (
                  <SettingRow label={t("admin.settings.fields.tls_mode")}>
                    <Badge variant="outline">
                      {settings.notifier.tls_mode}
                    </Badge>
                  </SettingRow>
                )}
                {settings.notifier.skip_verify_tls !== undefined && (
                  <SettingRow
                    label={t("admin.settings.fields.skip_verify_tls")}
                  >
                    <BoolValue
                      value={settings.notifier.skip_verify_tls}
                      insecureWhen={true}
                    />
                  </SettingRow>
                )}
                {settings.notifier.directory && (
                  <SettingRow label={t("admin.settings.fields.directory")}>
                    <TextValue value={settings.notifier.directory} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="activity"
                title={t("admin.settings.sections.activity")}
                icon={ScrollText}
              >
                <SettingRow label={t("admin.settings.fields.type")}>
                  <Badge variant="outline">{settings.activity.type}</Badge>
                </SettingRow>
                {settings.activity.endpoint && (
                  <SettingRow label={t("admin.settings.fields.endpoint")}>
                    <CopyableValue value={settings.activity.endpoint} />
                  </SettingRow>
                )}
                {settings.activity.directory && (
                  <SettingRow label={t("admin.settings.fields.directory")}>
                    <TextValue value={settings.activity.directory} />
                  </SettingRow>
                )}
              </SettingsSection>

              <SettingsSection
                id="auth"
                title={t("admin.settings.sections.auth")}
                description={t("admin.settings.sections.auth_description")}
                icon={ShieldCheck}
              >
                {settings.auth.providers.length === 0 ? (
                  <div className="py-2 text-sm text-muted-foreground">
                    {t("admin.settings.no_providers")}
                  </div>
                ) : (
                  settings.auth.providers.map((provider) => (
                    <AuthProviderBlock key={provider.key} provider={provider} />
                  ))
                )}
              </SettingsSection>

              <SettingsSection
                id="observability"
                title={t("admin.settings.sections.observability")}
                icon={Gauge}
              >
                <div className="space-y-1 py-3">
                  <div className="font-medium">
                    {t("admin.settings.fields.profiling")}
                  </div>
                  <SettingRow label={t("admin.settings.fields.enabled")}>
                    <BoolValue
                      value={settings.observability.profiling.enabled}
                    />
                  </SettingRow>
                  {settings.observability.profiling.enabled && (
                    <>
                      <SettingRow label={t("admin.settings.fields.type")}>
                        <TextValue
                          value={settings.observability.profiling.type}
                        />
                      </SettingRow>
                      <SettingRow
                        label={t("admin.settings.fields.server_address")}
                      >
                        <CopyableValue
                          value={
                            settings.observability.profiling.server_address
                          }
                        />
                      </SettingRow>
                      <SettingRow
                        label={t("admin.settings.fields.application_name")}
                      >
                        <TextValue
                          value={
                            settings.observability.profiling.application_name
                          }
                        />
                      </SettingRow>
                      <SettingRow
                        label={t("admin.settings.fields.upload_rate")}
                      >
                        <TextValue
                          value={settings.observability.profiling.upload_rate}
                        />
                      </SettingRow>
                    </>
                  )}
                </div>
                <div className="space-y-1 py-3">
                  <div className="font-medium">
                    {t("admin.settings.fields.tracing")}
                  </div>
                  <SettingRow label={t("admin.settings.fields.enabled")}>
                    <BoolValue value={settings.observability.tracing.enabled} />
                  </SettingRow>
                  {settings.observability.tracing.enabled && (
                    <>
                      <SettingRow label={t("admin.settings.fields.type")}>
                        <TextValue
                          value={settings.observability.tracing.type}
                        />
                      </SettingRow>
                      <SettingRow label={t("admin.settings.fields.endpoint")}>
                        <CopyableValue
                          value={settings.observability.tracing.endpoint}
                        />
                      </SettingRow>
                      <SettingRow
                        label={t("admin.settings.fields.service_name")}
                      >
                        <TextValue
                          value={settings.observability.tracing.service_name}
                        />
                      </SettingRow>
                      <SettingRow
                        label={t("admin.settings.fields.sampling_rate")}
                      >
                        <TextValue
                          value={settings.observability.tracing.sampling_rate}
                        />
                      </SettingRow>
                    </>
                  )}
                </div>
              </SettingsSection>

              <SettingsSection
                id="security"
                title={t("admin.settings.sections.security")}
                icon={Lock}
              >
                <SettingRow
                  label={t("admin.settings.fields.authenticated_rate_limit")}
                >
                  <RateValue
                    value={settings.security.authenticated_requests_per_minute}
                  />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.unauthenticated_rate_limit")}
                >
                  <RateValue
                    value={
                      settings.security.unauthenticated_requests_per_minute
                    }
                  />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.access_token_expiry")}
                >
                  <DurationValue
                    minutes={settings.security.access_token_expiry}
                  />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.refresh_token_expiry")}
                >
                  <DurationValue
                    minutes={settings.security.refresh_token_expiry}
                  />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.mfa_token_expiry")}>
                  <DurationValue minutes={settings.security.mfa_token_expiry} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.trusted_proxies")}>
                  <ListValue values={settings.security.trusted_proxies} />
                </SettingRow>
                <SettingRow label={t("admin.settings.fields.allowed_origins")}>
                  <ListValue values={settings.security.allowed_origins} />
                </SettingRow>
                <SettingRow
                  label={t("admin.settings.fields.cookie_secure_force")}
                >
                  <BoolValue
                    value={settings.security.cookie_secure_force}
                    insecureWhen={false}
                  />
                </SettingRow>
              </SettingsSection>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
