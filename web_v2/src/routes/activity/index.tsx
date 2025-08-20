import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { bucketActivityQueryOptions } from "@/queries/bucket.ts";

import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton.tsx";
import { Card, CardContent } from "@/components/ui/card";

export const Route = createFileRoute("/activity/")({
  loader: ({ context: { queryClient } }) =>
    queryClient.ensureQueryData(bucketActivityQueryOptions()),
  pendingComponent: ActivityViewSkeleton,
  component: ActivityPage,
});

function ActivityPage() {
  const activityQuery = useSuspenseQuery(bucketActivityQueryOptions());
  const activity = activityQuery.data;

  return (
    <div className="w-full flex-1 overflow-auto">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Activity Feed</h1>
        </div>

        <Card>
          <CardContent className="pb-0">
            <ActivityView activity={activity} />
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
