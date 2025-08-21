import React from "react";

import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton";
import { useBucketActivityData } from "@/components/bucket-view/hooks/useBucketActivityData";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { Card, CardContent } from "@/components/ui/card";

export const BucketActivityView = () => {
  const { bucketId } = useBucketViewContext();
  const { activity, isLoading } = useBucketActivityData(bucketId);

  return (
    <Card>
      <CardContent className="pb-0">
        {isLoading ? (
          <ActivityViewSkeleton />
        ) : (
          <ActivityView activity={activity} />
        )}
      </CardContent>
    </Card>
  );
};
