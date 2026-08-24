import { useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { DateRange } from "react-day-picker";
import type { FC } from "react";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton.tsx";
import {
  ActivityDateRangePicker,
  dateRangeToQuery,
} from "@/components/activity-view/components/ActivityDateRangePicker";
import { ActivityTimeline } from "@/components/bucket-view/components/ActivityTimeline";
import { Button } from "@/components/ui/button";
import { bucketActivityInfiniteQueryOptions } from "@/queries/bucket.ts";

export const ActivityTab: FC = () => {
  const { t } = useTranslation();
  const { bucketId } = useParams({
    from: "/_authenticated/buckets/$bucketId/activity",
  });
  const [dateRange, setDateRange] = useState<DateRange | undefined>();

  const {
    data: activity = [],
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    ...bucketActivityInfiniteQueryOptions(
      bucketId,
      dateRangeToQuery(dateRange),
    ),
    select: (data) => data.pages.flatMap((page) => page.data),
  });

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4">
      <div className="flex items-center justify-end">
        <ActivityDateRangePicker value={dateRange} onChange={setDateRange} />
      </div>

      <div className="bg-card border-border min-h-0 rounded-xl border p-6">
        {isLoading ? (
          <ActivityViewSkeleton />
        ) : (
          <>
            <ActivityTimeline activity={activity} />
            {hasNextPage && (
              <div className="flex justify-start pt-4 pl-2">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage
                    ? t("activity.loading_more")
                    : t("activity.load_more")}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
