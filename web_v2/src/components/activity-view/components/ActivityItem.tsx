import type { IActivity } from "@/types/activity.ts";

import { cn } from "@/lib/utils.ts";

import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar.tsx";
import {
  formatMessage,
  getActivityMapping,
  timeAgo,
} from "@/components/activity-view/helpers/utils.ts";

interface ActivityItemProps {
  item: IActivity;
}

export function ActivityItem({ item }: ActivityItemProps) {
  const { icon: Icon, iconColor, iconBg } = getActivityMapping(item.message);

  return (
    <div className="flex items-start space-x-4 py-4">
      <Avatar className="h-10 w-10">
        <AvatarImage src="/placeholder.svg" />
        <AvatarFallback>
          {item.user.email.charAt(0).toUpperCase()}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 space-y-1">
        <div className="flex items-center">
          <p className="font-medium">
            {item.user.first_name} {item.user.last_name}
          </p>
          <div
            className={cn(
              "ml-2 flex h-6 w-6 items-center justify-center rounded-full",
              iconBg,
            )}
          >
            <Icon className={cn("h-3.5 w-3.5", iconColor)} />
          </div>
        </div>
        <p className="text-muted-foreground text-sm">{formatMessage(item)}</p>
        <p className="text-muted-foreground text-xs">
          {timeAgo(item.timestamp)}
        </p>
      </div>
    </div>
  );
}
