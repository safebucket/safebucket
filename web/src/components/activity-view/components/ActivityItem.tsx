import { useTranslation } from "react-i18next";
import type { IActivity } from "@/types/activity.ts";

import { cn } from "@/lib/utils.ts";

import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar.tsx";
import {
  Item,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
} from "@/components/ui/item";
import {
  formatMessage,
  getActivityMapping,
  timeAgo,
} from "@/components/activity-view/helpers/utils.ts";

interface ActivityItemProps {
  item: IActivity;
}

export function ActivityItem({ item }: ActivityItemProps) {
  const { t } = useTranslation();
  const { icon: Icon, iconColor, iconBg } = getActivityMapping(item.message);

  return (
    <Item>
      <ItemMedia variant="image">
        <Avatar className="h-10 w-10">
          <AvatarImage src="/placeholder.svg" />
          <AvatarFallback>
            {item.user.email.charAt(0).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      </ItemMedia>
      <ItemContent>
        <ItemTitle>
          {item.user.first_name} {item.user.last_name}
          <div
            className={cn(
              "flex h-6 w-6 items-center justify-center rounded-full",
              iconBg,
            )}
          >
            <Icon className={cn("h-3.5 w-3.5", iconColor)} />
          </div>
        </ItemTitle>
        <ItemDescription>{formatMessage(item, t)}</ItemDescription>
        <p className="text-muted-foreground text-xs">
          {timeAgo(item.timestamp, t)}
        </p>
      </ItemContent>
    </Item>
  );
}
