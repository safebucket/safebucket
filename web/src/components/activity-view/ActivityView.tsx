import { useTranslation } from "react-i18next";
import type { IActivity } from "@/types/activity.ts";

import { ItemGroup, ItemSeparator } from "@/components/ui/item";
import { ActivityItem } from "@/components/activity-view/components/ActivityItem.tsx";

interface ActivityViewProps {
  activity: Array<IActivity>;
}

export function ActivityView({ activity }: ActivityViewProps) {
  const { t } = useTranslation();

  if (!activity.length) {
    return (
      <p className="flex h-24 items-center justify-center text-center">
        {t("activity.no_activity_yet")}
      </p>
    );
  }

  return (
    <ItemGroup>
      {activity.map((item, index) => (
        <div key={index}>
          <ActivityItem item={item} />
          {index < activity.length - 1 && <ItemSeparator />}
        </div>
      ))}
    </ItemGroup>
  );
}
