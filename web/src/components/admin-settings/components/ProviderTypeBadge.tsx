import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

export function ProviderTypeBadge({ type }: { type: string }) {
  const { t } = useTranslation();

  const label =
    type === "local"
      ? t("admin.settings.provider_types.local")
      : type === "oidc"
        ? t("admin.settings.provider_types.oidc")
        : type === "ldap"
          ? t("admin.settings.provider_types.ldap")
          : type;

  return <Badge variant="outline">{label}</Badge>;
}
