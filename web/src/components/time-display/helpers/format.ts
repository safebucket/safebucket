import type { TimeDisplayMode } from "@/components/time-display/context/TimeDisplayProvider";

const zoneFor = (mode: TimeDisplayMode): string | undefined =>
  mode === "utc" ? "UTC" : undefined;

export function formatAbsoluteTimestamp(
  nanoTimestamp: string,
  mode: TimeDisplayMode,
  locale?: string,
): string {
  const ms = Number(nanoTimestamp) / 1000000;
  if (!Number.isFinite(ms)) {
    return "-";
  }

  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZoneName: "short",
    timeZone: zoneFor(mode),
  }).format(new Date(ms));
}

export function zoneDayKey(date: Date, mode: TimeDisplayMode): string {
  const parts = new Intl.DateTimeFormat("en-US", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    timeZone: zoneFor(mode),
  }).formatToParts(date);

  const value = (type: string): string =>
    parts.find((part) => part.type === type)?.value ?? "";

  return `${value("year")}-${value("month")}-${value("day")}`;
}

export function formatDayLabel(dayKey: string, locale?: string): string {
  const [year, month, day] = dayKey.split("-").map(Number);
  if (!year || !month || !day) {
    return dayKey;
  }

  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
  }).format(new Date(year, month - 1, day));
}
