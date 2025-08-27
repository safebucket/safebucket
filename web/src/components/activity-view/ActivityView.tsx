import { useTranslation } from "react-i18next";
import type { IActivity } from "@/types/activity.ts";

import { Separator } from "@/components/ui/separator";
import { ActivityItem } from "@/components/activity-view/components/ActivityItem.tsx";

interface ActivityViewProps {
  activity: Array<IActivity>;
}

export function ActivityView({ activity }: ActivityViewProps) {
  const { t } = useTranslation();

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
          {t("activity.no_activity_yet")}
        </p>
      )}
    </>
  );
}
