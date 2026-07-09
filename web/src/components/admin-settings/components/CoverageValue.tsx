import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";

export function CoverageValue({ covered }: { covered: boolean }) {
  const { t } = useTranslation();

  return (
    <Badge variant={covered ? "default" : "secondary"}>
      {covered
        ? t("admin.settings.values.enabled")
        : t("admin.settings.values.disabled")}
    </Badge>
  );
}
