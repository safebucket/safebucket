import { TriangleAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

export function BoolValue({
  value,
  insecureWhen,
}: {
  value: boolean;
  insecureWhen?: boolean;
}) {
  const { t } = useTranslation();
  const isInsecure = insecureWhen !== undefined && value === insecureWhen;

  return (
    <Badge
      variant={isInsecure ? "destructive" : value ? "default" : "secondary"}
      title={isInsecure ? t("admin.settings.insecure") : undefined}
    >
      {isInsecure && <TriangleAlert />}
      {value ? t("admin.settings.values.yes") : t("admin.settings.values.no")}
    </Badge>
  );
}
