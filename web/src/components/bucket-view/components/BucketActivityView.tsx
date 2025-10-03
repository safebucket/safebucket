import { useQuery } from "@tanstack/react-query";
import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton.tsx";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { Card, CardContent } from "@/components/ui/card";
import { bucketActivityQueryOptions } from "@/queries/bucket.ts";

export const BucketActivityView = () => {
  const { bucketId } = useBucketViewContext();

  const { data: activity, isLoading } = useQuery(
    bucketActivityQueryOptions(bucketId),
  );

  return (
    <Card className="py-2">
      <CardContent className="pb-0 px-2">
        {isLoading ? (
          <ActivityViewSkeleton />
        ) : (
          <ActivityView activity={activity!} />
        )}
      </CardContent>
    </Card>
  );
};
