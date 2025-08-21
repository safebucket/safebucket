import React, { FC } from "react";

import { ActivityItem } from "@/components/activity-view/components/ActivityItem";
import { IActivity } from "@/components/common/types/activity";
import { Separator } from "@/components/ui/separator";

interface IActivityViewProps {
  activity: IActivity[];
}

export const ActivityView: FC<IActivityViewProps> = ({
  activity,
}: IActivityViewProps) => (
  <>
    {activity.map((item, index) => (
      <div key={index}>
        <ActivityItem item={item} />
        {index < activity.length - 1 && <Separator />}
      </div>
    ))}

    {!activity.length && (
      <p className="flex h-24 items-center justify-center text-center">
        No activity yet.
      </p>
    )}
  </>
);
