import { AlertCircle } from "lucide-react";

import type { ActivityMessage, IActivity } from "@/types/activity.ts";
import type { IMessageMapping } from "@/components/activity-view/helpers/types.ts";
import { messageMap } from "@/components/activity-view/helpers/constants";

const DEFAULT_ACTIVITY_MAPPING = {
  messageKey: "activity.messages.default",
  icon: AlertCircle,
  iconColor: "text-gray-500",
  iconBg: "bg-gray-100",
};

export function getActivityMapping(
  messageType: ActivityMessage,
): IMessageMapping {
  return messageMap[messageType] || DEFAULT_ACTIVITY_MAPPING;
}

export const formatMessage = (
  log: IActivity,
  t: (key: string) => string,
): string => {
  const mapping = messageMap[log.message] || DEFAULT_ACTIVITY_MAPPING;
  return t(mapping.messageKey)
    .replace("%%USERNAME%%", `${log.user.first_name} ${log.user.last_name}`)
    .replace("%%BUCKET_NAME%%", log.bucket?.name || "")
    .replace("%%FILE_NAME%%", log.file?.name || "")
    .replace("%%BUCKET_MEMBER_EMAIL%%", log.bucket_member_email || "");
};

export const timeAgo = (
  nanoTimestamp: string,
  t: (key: string, options?: any) => string,
): string => {
  const nanoseconds = Number(nanoTimestamp);
  const timestampMs = nanoseconds / 1000000;

  const now = Date.now();

  const diffMs = now - timestampMs;

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(diffMs / (1000 * 60));
  const hours = Math.floor(diffMs / (1000 * 60 * 60));
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  const weeks = Math.floor(days / 7);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  const format = (count: number, unit: string) => {
    const unitKey =
      count === 1 ? `activity.time.${unit}` : `activity.time.${unit}s`;
    return `${t("activity.time.ago", { value: count, unit: t(unitKey) })}`;
  };

  if (seconds < 60) return format(seconds, "second");
  if (minutes < 60) return format(minutes, "minute");
  if (hours < 24) return format(hours, "hour");
  if (days < 7) return format(days, "day");
  if (weeks < 4) return format(weeks, "week");
  if (months < 12) return format(months, "month");
  return format(years, "year");
};
