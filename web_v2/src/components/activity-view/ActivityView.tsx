import type { Activity } from "@/types/activity";

import { Separator } from "@/components/ui/separator";

import { ActivityItem } from "./ActivityItem";

interface ActivityViewProps {
  activity: Activity[];
}

export function ActivityView({ activity }: ActivityViewProps) {
  return (
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
}
