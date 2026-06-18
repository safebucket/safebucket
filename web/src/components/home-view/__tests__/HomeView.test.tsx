// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render } from "@testing-library/react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { HomeView } from "../HomeView";
import { useCurrentUser, useUserStatsQuery } from "@/queries/user";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { firstName?: string }) =>
      opts?.firstName ? `${key} ${opts.firstName}` : key,
  }),
}));

vi.mock("@tanstack/react-query", () => ({
  useInfiniteQuery: vi.fn(),
}));

vi.mock("@/queries/user", () => ({
  useCurrentUser: vi.fn(),
  useUserStatsQuery: vi.fn(),
}));

vi.mock("@/queries/bucket", () => ({
  bucketsActivityInfiniteQueryOptions: () => ({ queryKey: ["activity"] }),
}));

vi.mock("@/components/activity-view/components/ActivityItem.tsx", () => ({
  ActivityItem: () => <div data-testid="activity-item" />,
}));

vi.mock(
  "@/components/activity-view/components/ActivityViewSkeleton.tsx",
  () => ({
    ActivityViewSkeleton: () => <div data-testid="activity-skeleton" />,
  }),
);

const mockUseInfiniteQuery = vi.mocked(useInfiniteQuery);
const mockUseCurrentUser = vi.mocked(useCurrentUser);
const mockUseUserStatsQuery = vi.mocked(useUserStatsQuery);

const activity = (data: Array<unknown>, isLoading = false) =>
  ({ data, isLoading }) as unknown as ReturnType<typeof useInfiniteQuery>;

const stats = (
  data: { total_files: number; total_buckets: number } | undefined,
  isLoading = false,
) => ({ data, isLoading }) as unknown as ReturnType<typeof useUserStatsQuery>;

beforeEach(() => {
  mockUseInfiniteQuery.mockReturnValue(activity([]));
  mockUseCurrentUser.mockReturnValue({
    data: { id: "u1", first_name: "Jane" },
  } as unknown as ReturnType<typeof useCurrentUser>);
  mockUseUserStatsQuery.mockReturnValue(
    stats({ total_files: 12, total_buckets: 9 }),
  );
});

afterEach(cleanup);

describe("HomeView", () => {
  it("keeps its content in a vertically scrollable container so mobile content stays reachable", () => {
    const { container } = render(<HomeView />);
    const root = container.firstElementChild as HTMLElement;

    expect(root.className).toContain("overflow-y-auto");
    expect(root.className).toContain("flex-1");
    expect(root.className).toContain("min-h-0");
  });

  it("greets the signed-in user by first name", () => {
    const { getByRole } = render(<HomeView />);

    expect(getByRole("heading", { level: 1 }).textContent).toContain("Jane");
  });

  it("shows the user statistics returned by the query", () => {
    const { getByText } = render(<HomeView />);

    expect(getByText("12")).toBeTruthy();
    expect(getByText("9")).toBeTruthy();
  });

  it("falls back to zero when the statistics are missing", () => {
    mockUseUserStatsQuery.mockReturnValue(stats(undefined));

    const { getAllByText } = render(<HomeView />);

    expect(getAllByText("0")).toHaveLength(2);
  });

  it("shows skeletons instead of numbers while statistics are loading", () => {
    mockUseUserStatsQuery.mockReturnValue(stats(undefined, true));

    const { container, queryByText } = render(<HomeView />);

    expect(container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(
      2,
    );
    expect(queryByText("12")).toBeNull();
  });

  it("renders at most the three most recent activity entries", () => {
    mockUseInfiniteQuery.mockReturnValue(activity([{}, {}, {}, {}, {}]));

    const { getAllByTestId } = render(<HomeView />);

    expect(getAllByTestId("activity-item")).toHaveLength(3);
  });

  it("shows an empty state when there is no activity", () => {
    const { getByText, queryByTestId } = render(<HomeView />);

    expect(getByText("activity.no_activity_yet")).toBeTruthy();
    expect(queryByTestId("activity-item")).toBeNull();
  });

  it("shows the activity skeleton while activity is loading", () => {
    mockUseInfiniteQuery.mockReturnValue(activity([], true));

    const { getByTestId, queryByTestId } = render(<HomeView />);

    expect(getByTestId("activity-skeleton")).toBeTruthy();
    expect(queryByTestId("activity-item")).toBeNull();
  });

  it("opens the documentation site in a new tab from the quick start card", () => {
    const openSpy = vi.spyOn(window, "open").mockImplementation(() => null);
    const { getByRole } = render(<HomeView />);

    fireEvent.click(
      getByRole("button", {
        name: "homepage.quick_start.view_documentation",
      }),
    );

    expect(openSpy).toHaveBeenCalledWith(
      "https://docs.safebucket.io",
      "_blank",
    );
    openSpy.mockRestore();
  });
});
