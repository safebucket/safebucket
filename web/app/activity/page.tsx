"use client";

import React from "react";

import { ActivityView } from "@/components/activity-view/ActivityView";
import { ActivityViewSkeleton } from "@/components/activity-view/components/ActivityViewSkeleton";
import { useActivityData } from "@/components/activity-view/hooks/useActivityData";
import { Card, CardContent } from "@/components/ui/card";

export default function ActivityFeed() {
  const { activity, isLoading } = useActivityData();

  return (
    <div className="flex-1 w-full overflow-auto">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Activity Feed</h1>
        </div>

        <Card>
          <CardContent className="pb-0">
            {isLoading ? (
              <ActivityViewSkeleton />
            ) : (
              <ActivityView activity={activity} />
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
