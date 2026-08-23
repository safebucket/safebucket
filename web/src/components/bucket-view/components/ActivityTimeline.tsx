import { Link2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IActivity } from "@/types/activity.ts";
import {
  formatMessage,
  getActivityMapping,
  getUserDisplayName,
  timeAgo,
} from "@/components/activity-view/helpers/utils.ts";
import { formatAbsoluteTimestamp } from "@/components/time-display/helpers/format";
import { useTimeDisplay } from "@/components/time-display/hooks/useTimeDisplay";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";

interface IActivityTimelineProps {
  activity: Array<IActivity>;
}

export const ActivityTimeline: FC<IActivityTimelineProps> = ({
  activity,
}: IActivityTimelineProps) => {
  const { t, i18n } = useTranslation();
  const { mode } = useTimeDisplay();

  if (!activity.length) {
    return (
      <p className="text-muted-foreground flex h-24 items-center justify-center text-center">
        {t("activity.no_activity_yet")}
      </p>
    );
  }

  return (
    <ol className="m-0 flex list-none flex-col p-0 pt-1 pl-2">
      {activity.map((item, index) => {
        const isLast = index === activity.length - 1;
        const {
          icon: Icon,
          iconColor,
          iconBg,
        } = getActivityMapping(item.message);
        return (
          <li
            key={index}
            className="relative ms-8 flex flex-1 flex-col gap-1 [&:not(:last-child)]:pb-6"
          >
            {!isLast ? (
              <div className="bg-border absolute top-9 -left-6 h-[calc(100%-1.75rem)] w-0.5 -translate-x-1/2" />
            ) : null}

            <div className="absolute top-0 -left-6 z-10 -translate-x-1/2">
              <Avatar className="size-8">
                <AvatarFallback>
                  {item.user ? (
                    item.user.email.charAt(0).toUpperCase()
                  ) : (
                    <Link2 className="h-4 w-4" />
                  )}
                </AvatarFallback>
              </Avatar>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">
                {getUserDisplayName(item.user, t("activity.share_link"))}
              </span>
              <div
                className={cn(
                  "flex h-5 w-5 items-center justify-center rounded-full",
                  iconBg,
                )}
              >
                <Icon className={cn("h-3 w-3", iconColor)} />
              </div>
              <time
                className="text-muted-foreground ml-auto text-xs"
                title={formatAbsoluteTimestamp(
                  item.timestamp,
                  mode,
                  i18n.language,
                )}
              >
                {timeAgo(item.timestamp, t)}
              </time>
            </div>

            <p className="text-muted-foreground text-sm">
              {formatMessage(item, t)}
            </p>
          </li>
        );
      })}
    </ol>
  );
};
