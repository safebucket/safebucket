import { useState } from "react";
import { useInfiniteQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import type { DateRange } from "react-day-picker";
import { bucketsActivityInfiniteQueryOptions } from "@/queries/bucket.ts";

import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityPageSkeleton } from "@/components/activity-view/components/ActivityPageSkeleton.tsx";
import {
  ActivityDateRangePicker,
  dateRangeToQuery,
} from "@/components/activity-view/components/ActivityDateRangePicker";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

export const Route = createFileRoute("/_authenticated/activity/")({
  component: ActivityPage,
});

function ActivityPage() {
  const { t } = useTranslation();
  const [dateRange, setDateRange] = useState<DateRange | undefined>();

  const {
    data: activity = [],
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfiniteQuery({
    ...bucketsActivityInfiniteQueryOptions(dateRangeToQuery(dateRange)),
    select: (data) => data.pages.flatMap((page) => page.data),
  });

  return (
    <div className="flex w-full min-h-0 flex-1 flex-col">
      <div className="mx-6 flex min-h-0 flex-1 flex-col gap-8">
        <div className="shrink-0 flex items-center justify-between">
          <h1 className="text-2xl font-bold">{t("activity.title")}</h1>
          <ActivityDateRangePicker value={dateRange} onChange={setDateRange} />
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <Card className="mb-6 py-2">
            <CardContent className="pb-0 px-2">
              {isLoading ? (
                <ActivityPageSkeleton />
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
      </div>
    </div>
  );
}
