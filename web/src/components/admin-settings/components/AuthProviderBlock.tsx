import { useTranslation } from "react-i18next";
import { SettingRow } from "./SettingRow";
import { BoolValue } from "./BoolValue";
import { CopyableValue } from "./CopyableValue";
import { ListValue } from "./ListValue";
import { ProviderTypeBadge } from "./ProviderTypeBadge";
import { TextValue } from "./TextValue";
import type { AdminAuthProviderSettings } from "@/types/admin";

export function AuthProviderBlock({
  provider,
}: {
  provider: AdminAuthProviderSettings;
}) {
  const { t } = useTranslation();

  return (
    <div className="space-y-1 py-3">
      <div className="flex items-center gap-2 font-medium">
        {provider.name || provider.key}
        <ProviderTypeBadge type={provider.type} />
      </div>
      <SettingRow label={t("admin.settings.fields.domains")}>
        <ListValue values={provider.domains} />
      </SettingRow>
      <SettingRow label={t("admin.settings.fields.mfa_required")}>
        <BoolValue value={provider.mfa_required} />
      </SettingRow>
      <SettingRow label={t("admin.settings.fields.sharing_allowed")}>
        <BoolValue value={provider.sharing_allowed} />
      </SettingRow>
      {provider.issuer && (
        <SettingRow label={t("admin.settings.fields.issuer")}>
          <CopyableValue value={provider.issuer} />
        </SettingRow>
      )}
      {provider.url && (
        <SettingRow label={t("admin.settings.fields.url")}>
          <CopyableValue value={provider.url} />
        </SettingRow>
      )}
      {provider.base_dn && (
        <SettingRow label={t("admin.settings.fields.base_dn")}>
          <TextValue value={provider.base_dn} />
        </SettingRow>
      )}
      {provider.user_filter && (
        <SettingRow label={t("admin.settings.fields.user_filter")}>
          <TextValue value={provider.user_filter} />
        </SettingRow>
      )}
      {provider.attribute_email && (
        <SettingRow label={t("admin.settings.fields.attribute_email")}>
          <TextValue value={provider.attribute_email} />
        </SettingRow>
      )}
      {provider.start_tls !== undefined && (
        <SettingRow label={t("admin.settings.fields.start_tls")}>
          <BoolValue value={provider.start_tls} />
        </SettingRow>
      )}
      {provider.tls_insecure_skip !== undefined && (
        <SettingRow label={t("admin.settings.fields.tls_insecure_skip")}>
          <BoolValue value={provider.tls_insecure_skip} insecureWhen={true} />
        </SettingRow>
      )}
    </div>
  );
}
