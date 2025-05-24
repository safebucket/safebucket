import React, { FC } from "react";

import { IActivity } from "@/components/activity-view/helpers/types";
import { ActivityItem } from "@/components/activity-view/components/ActivityItem";
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
      <p className="flex items-center justify-center h-24 text-center">
        No activity yet.
      </p>
    )}
  </>
);
