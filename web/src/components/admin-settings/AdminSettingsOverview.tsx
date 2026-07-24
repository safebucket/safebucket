import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import {
  Boxes,
  Cpu,
  Database,
  HardDrive,
  Mail,
  Radio,
  ScrollText,
  Server,
  ShieldCheck,
  SlidersHorizontal,
} from "lucide-react";
import { useAdminSettings } from "./hooks/useAdminSettings";
import { SettingsSection } from "./components/SettingsSection";
import { SettingRow } from "./components/SettingRow";
import { SettingsError } from "./components/SettingsError";
import { OverviewCard } from "./components/OverviewCard";
import { CoverageValue } from "./components/CoverageValue";
import { ProviderTypeBadge } from "./components/ProviderTypeBadge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

export function AdminSettingsOverview() {
  const { t } = useTranslation();
  const { data: settings, isLoading, isError, refetch } = useAdminSettings();

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="container mx-auto max-w-3xl p-6">
        <div className="mb-6 flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold">
              {t("admin.settings.title")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("admin.settings.description")}
            </p>
          </div>
          <Button asChild variant="outline" size="sm">
            <Link to="/admin/settings/details">
              {t("admin.settings.view_all")}
            </Link>
          </Button>
        </div>

        {isError ? (
          <SettingsError onRetry={() => void refetch()} />
        ) : isLoading || !settings ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, index) => (
              <Skeleton key={index} className="h-24 w-full" />
            ))}
          </div>
        ) : (
          <div className="space-y-6">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <OverviewCard
                title={t("admin.settings.fields.platforms")}
                value={
                  settings.platforms === null
                    ? t("admin.settings.coverage.unknown")
                    : String(settings.platforms)
                }
                icon={Server}
              />
              <OverviewCard
                title={t("admin.settings.sections.app")}
                value={settings.app.profile}
                hint={settings.app.log_level}
                icon={SlidersHorizontal}
              />
              <OverviewCard
                title={t("admin.settings.sections.storage")}
                value={settings.storage.type}
                hint={settings.storage.bucket_name}
                icon={HardDrive}
              />
              <OverviewCard
                title={t("admin.settings.sections.database")}
                value={settings.database.type}
                hint={settings.database.host ?? settings.database.path}
                icon={Database}
              />
              <OverviewCard
                title={t("admin.settings.sections.cache")}
                value={settings.cache.type}
                hint={settings.cache.hosts?.[0]}
                icon={Boxes}
              />
              <OverviewCard
                title={t("admin.settings.sections.events")}
                value={settings.events.type}
                hint={settings.events.host}
                icon={Radio}
              />
              <OverviewCard
                title={t("admin.settings.sections.notifier")}
                value={settings.notifier.type}
                hint={settings.notifier.host}
                icon={Mail}
              />
              <OverviewCard
                title={t("admin.settings.sections.activity")}
                value={settings.activity.type}
                hint={settings.activity.endpoint ?? settings.activity.directory}
                icon={ScrollText}
              />
              <OverviewCard
                title={t("admin.settings.sections.auth")}
                value={String(settings.auth.providers.length)}
                icon={ShieldCheck}
              />
            </div>

            <SettingsSection
              title={t("admin.settings.sections.workers")}
              description={t("admin.settings.sections.workers_description")}
              icon={Cpu}
            >
              <SettingRow label={t("admin.settings.fields.http_server")}>
                <CoverageValue status={settings.workers.http_server} />
              </SettingRow>
              <SettingRow label={t("admin.settings.fields.object_deletion")}>
                <CoverageValue status={settings.workers.object_deletion} />
              </SettingRow>
              <SettingRow label={t("admin.settings.fields.bucket_events")}>
                <CoverageValue status={settings.workers.bucket_events} />
              </SettingRow>
              <SettingRow label={t("admin.settings.fields.trash_cleanup")}>
                <CoverageValue status={settings.workers.trash_cleanup} />
              </SettingRow>
              <SettingRow label={t("admin.settings.fields.garbage_collector")}>
                <CoverageValue status={settings.workers.garbage_collector} />
              </SettingRow>
            </SettingsSection>

            <SettingsSection
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
                  <SettingRow
                    key={provider.key}
                    label={provider.name || provider.key}
                  >
                    <ProviderTypeBadge type={provider.type} />
                  </SettingRow>
                ))
              )}
            </SettingsSection>
          </div>
        )}
      </div>
    </div>
  );
}
