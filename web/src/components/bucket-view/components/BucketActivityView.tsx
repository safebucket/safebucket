import { useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { DateRange } from "react-day-picker";
import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton.tsx";
import {
  ActivityDateRangePicker,
  dateRangeToQuery,
} from "@/components/activity-view/components/ActivityDateRangePicker";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { bucketActivityInfiniteQueryOptions } from "@/queries/bucket.ts";

export const BucketActivityView = () => {
  const { t } = useTranslation();
  const { bucketId } = useBucketViewContext();
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
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div className="flex justify-end">
        <ActivityDateRangePicker value={dateRange} onChange={setDateRange} />
      </div>
      <Card className="mb-6 flex min-h-0 flex-1 flex-col py-2">
        <CardContent className="min-h-0 overflow-y-auto pb-0 px-2">
          {isLoading ? (
            <ActivityViewSkeleton />
          ) : (
            <>
              <ActivityView activity={activity} />
              {hasNextPage && (
                <div className="flex justify-center py-4">
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
        </CardContent>
      </Card>
    </div>
  );
};
