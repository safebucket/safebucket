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
    <div className="w-full flex-1 overflow-auto">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">{t("activity.title")}</h1>
        </div>

        <Card className="py-2">
          <CardContent className="pb-0 px-2">
            <ActivityView activity={activity} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
