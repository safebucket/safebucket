import { describe, expect, it } from "vitest";
import { rollupHourlyToDaily } from "./rollup";
import type { TimeSeriesPoint } from "@/types/admin.ts";

const points: Array<TimeSeriesPoint> = [
  { timestamp: "2026-06-16T08:00:00Z", count: 1 },
  { timestamp: "2026-06-16T23:00:00Z", count: 2 },
  { timestamp: "2026-06-17T00:00:00Z", count: 3 },
];

describe("rollupHourlyToDaily", () => {
  it("buckets hourly counts into UTC days", () => {
    expect(rollupHourlyToDaily(points, "utc")).toEqual([
      { date: "2026-06-16", count: 3 },
      { date: "2026-06-17", count: 3 },
    ]);
  });

  it("preserves the total count regardless of mode", () => {
    const total = points.reduce((sum, point) => sum + point.count, 0);
    for (const mode of ["utc", "local"] as const) {
      const rolled = rollupHourlyToDaily(points, mode).reduce(
        (sum, point) => sum + point.count,
        0,
      );
      expect(rolled).toBe(total);
    }
  });

  it("skips invalid timestamps", () => {
    expect(
      rollupHourlyToDaily([{ timestamp: "not-a-date", count: 5 }], "utc"),
    ).toEqual([]);
  });
});
