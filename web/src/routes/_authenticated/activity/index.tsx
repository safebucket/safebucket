import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { bucketsActivityQueryOptions } from "@/queries/bucket.ts";

import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityPageSkeleton } from "@/components/activity-view/components/ActivityPageSkeleton.tsx";
import { Card, CardContent } from "@/components/ui/card";

export const Route = createFileRoute("/_authenticated/activity/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(bucketsActivityQueryOptions()),
  pendingComponent: ActivityPageSkeleton,
  component: ActivityPage,
});

function ActivityPage() {
  const activityQuery = useSuspenseQuery(bucketsActivityQueryOptions());
  const activity = activityQuery.data;

  const { t } = useTranslation();

  return (
    <div className="flex w-full min-h-0 flex-1 flex-col">
      <div className="mx-6 flex min-h-0 flex-1 flex-col gap-8">
        <div className="shrink-0 flex items-center justify-between">
          <h1 className="text-2xl font-bold">{t("activity.title")}</h1>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <Card className="py-2">
            <CardContent className="pb-0 px-2">
              <ActivityView activity={activity} />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
