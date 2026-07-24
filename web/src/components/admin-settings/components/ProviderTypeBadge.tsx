import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

const labelKeys: Record<string, string> = {
  local: "admin.settings.provider_types.local",
  oidc: "admin.settings.provider_types.oidc",
  ldap: "admin.settings.provider_types.ldap",
};

export function ProviderTypeBadge({ type }: { type: string }) {
  const { t } = useTranslation();
  const label = labelKeys[type] ? t(labelKeys[type]) : type;

  return <Badge variant="outline">{label}</Badge>;
}
