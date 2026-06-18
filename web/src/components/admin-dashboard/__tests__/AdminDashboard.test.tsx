// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render } from "@testing-library/react";
import { AdminDashboard } from "../AdminDashboard";
import { useAdminDashboardData } from "../hooks/useAdminDashboardData";
import type { AdminStatsResponse } from "@/types/admin.ts";
import type { ReactNode } from "react";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("../hooks/useAdminDashboardData", () => ({
  useAdminDashboardData: vi.fn(),
}));

vi.mock("../components/SharedFilesChart", () => ({
  SharedFilesChart: ({
    data,
    timeRange,
  }: {
    data: Array<unknown>;
    timeRange: string;
  }) => (
    <div
      data-testid="shared-files-chart"
      data-points={data.length}
      data-range={timeRange}
    />
  ),
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ to, children }: { to: string; children: ReactNode }) => (
    <a href={to}>{children}</a>
  ),
}));

const mockUseAdminDashboardData = vi.mocked(useAdminDashboardData);

const loaded = (overrides: Partial<AdminStatsResponse> = {}) =>
  mockUseAdminDashboardData.mockReturnValue({
    stats: {
      total_users: 5,
      total_buckets: 8,
      total_files: 20,
      total_folders: 4,
      total_storage: 1500,
      shared_files_per_hour: [
        { timestamp: "2026-06-16T08:00:00Z", count: 1 },
        { timestamp: "2026-06-16T09:00:00Z", count: 2 },
      ],
      ...overrides,
    },
    isLoading: false,
  });

beforeEach(() => {
  loaded();
});

afterEach(cleanup);

describe("AdminDashboard", () => {
  it("keeps the loaded content in a vertically scrollable container", () => {
    const { container } = render(<AdminDashboard />);
    const root = container.firstElementChild as HTMLElement;

    expect(root.className).toContain("overflow-y-auto");
    expect(root.className).toContain("flex-1");
    expect(root.className).toContain("min-h-0");
  });

  it("keeps the loading state in a vertically scrollable container and hides the chart", () => {
    mockUseAdminDashboardData.mockReturnValue({
      stats: undefined,
      isLoading: true,
    });

    const { container, queryByTestId } = render(<AdminDashboard />);
    const root = container.firstElementChild as HTMLElement;

    expect(root.className).toContain("overflow-y-auto");
    expect(
      container.querySelectorAll('[data-slot="skeleton"]').length,
    ).toBeGreaterThan(0);
    expect(queryByTestId("shared-files-chart")).toBeNull();
  });

  it("renders each stat including human-readable storage", () => {
    const { getByText } = render(<AdminDashboard />);

    expect(getByText("5")).toBeTruthy();
    expect(getByText("8")).toBeTruthy();
    expect(getByText("20")).toBeTruthy();
    expect(getByText("1.50 KB")).toBeTruthy();
  });

  it("links only the users and buckets stat cards to their admin pages", () => {
    const { container } = render(<AdminDashboard />);
    const hrefs = Array.from(container.querySelectorAll("a")).map((a) =>
      a.getAttribute("href"),
    );

    expect(hrefs).toEqual(["/admin/users", "/admin/buckets"]);
  });

  it("passes the activity data and selected time range to the chart", () => {
    const { getByTestId } = render(<AdminDashboard />);
    const chart = getByTestId("shared-files-chart");

    expect(chart.getAttribute("data-points")).toBe("2");
    expect(chart.getAttribute("data-range")).toBe("90");
  });

  it("falls back to zero for missing counts", () => {
    mockUseAdminDashboardData.mockReturnValue({
      stats: undefined,
      isLoading: false,
    });

    const { getAllByText, getByText } = render(<AdminDashboard />);

    expect(getAllByText("0")).toHaveLength(3);
    expect(getByText("0 Bytes")).toBeTruthy();
  });
});
