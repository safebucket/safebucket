import { useTranslation } from "react-i18next";
import { NotSet } from "./NotSet";
import { formatDurationParts } from "@/lib/utils";

export function DurationValue({ minutes }: { minutes?: number | null }) {
  const { t } = useTranslation();
  if (minutes === undefined || minutes === null) {
    return <NotSet />;
  }
  const { value, unit } = formatDurationParts(minutes);
  const label =
    unit === "days"
      ? t("admin.settings.values.duration_days", { value })
      : unit === "hours"
        ? t("admin.settings.values.duration_hours", { value })
        : t("admin.settings.values.duration_minutes", { value });
  return <span>{label}</span>;
}
