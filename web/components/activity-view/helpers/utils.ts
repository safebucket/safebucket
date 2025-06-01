import { messageMap } from "@/components/activity-view/helpers/constants";
import { IMessageMapping } from "@/components/activity-view/helpers/types";
import { ActivityMessage, IActivity } from "@/components/common/types/activity";

export function getActivityMapping(
  messageType: ActivityMessage,
): IMessageMapping {
  return messageMap[messageType];
}

export const formatMessage = (log: IActivity): string => {
  return messageMap[log.message].message
    .replace("%%USERNAME%%", `${log.user.first_name} ${log.user.last_name}`)
    .replace("%%BUCKET_NAME%%", log.bucket?.name || "")
    .replace("%%FILE_NAME%%", log.file?.name || "");
};

export const timeAgo = (nanoTimestamp: string): string => {
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

  const format = (count: number, unit: string) =>
    `${count} ${unit}${count === 1 ? "" : "s"} ago`;

  if (seconds < 60) return format(seconds, "second");
  if (minutes < 60) return format(minutes, "minute");
  if (hours < 24) return format(hours, "hour");
  if (days < 7) return format(days, "day");
  if (weeks < 4) return format(weeks, "week");
  if (months < 12) return format(months, "month");
  return format(years, "year");
};
