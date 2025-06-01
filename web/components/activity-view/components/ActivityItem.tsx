import React, { FC } from "react";

import { cn } from "@/lib/utils";

import { formatMessage, getActivityMapping, timeAgo } from "@/components/activity-view/helpers/utils";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { IActivity } from "@/components/common/types/activity";

interface IActivityItemProps {
  item: IActivity;
}

export const ActivityItem: FC<IActivityItemProps> = ({
  item,
}: IActivityItemProps) => {
  const { icon: Icon, iconColor, iconBg } = getActivityMapping(item.message);

  return (
    <div className="flex items-start space-x-4 py-4">
      <Avatar className="h-10 w-10">
        <AvatarImage src={"/placeholder.svg"} />
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
        <p className="text-sm text-muted-foreground">{formatMessage(item)}</p>
        <p className="text-xs text-muted-foreground">
          {timeAgo(item.timestamp)}
        </p>
      </div>
    </div>
  );
};
