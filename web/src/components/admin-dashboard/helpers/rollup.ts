import type { TimeDisplayMode } from "@/components/time-display/context/TimeDisplayProvider";
import type { TimeSeriesPoint } from "@/types/admin.ts";
import { zoneDayKey } from "@/components/time-display/helpers/format";

export interface DailyCount {
  date: string;
  count: number;
}

export function rollupHourlyToDaily(
  points: Array<TimeSeriesPoint>,
  mode: TimeDisplayMode,
): Array<DailyCount> {
  const totals = new Map<string, number>();

  for (const point of points) {
    const date = new Date(point.timestamp);
    if (Number.isNaN(date.getTime())) {
      continue;
    }
    const key = zoneDayKey(date, mode);
    totals.set(key, (totals.get(key) ?? 0) + point.count);
  }

  return Array.from(totals.entries())
    .map(([date, count]) => ({ date, count }))
    .sort((a, b) => a.date.localeCompare(b.date));
}
