import { useTranslation } from "react-i18next";
import { NotSet } from "./NotSet";
import { formatDurationParts } from "@/lib/utils";

const durationKeys: Record<"days" | "hours" | "minutes", string> = {
  days: "admin.settings.values.duration_days",
  hours: "admin.settings.values.duration_hours",
  minutes: "admin.settings.values.duration_minutes",
};

export function DurationValue({ minutes }: { minutes?: number | null }) {
  const { t } = useTranslation();
  if (minutes === undefined || minutes === null) {
    return <NotSet />;
  }
  const { value, unit } = formatDurationParts(minutes);
  const label = t(durationKeys[unit], { value });
  return <span>{label}</span>;
}
