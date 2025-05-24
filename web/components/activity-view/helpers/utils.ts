import { messageMap } from "@/components/activity-view/helpers/constants";
import {
  IMessageMapping,
} from "@/components/activity-view/helpers/types";
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

export const timeAgo = (isoTimestamp: string): string => {
  const now = new Date();
  const then = new Date(isoTimestamp);

  const diffMs = now.getTime() - then.getTime();
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return "just now";
  if (diffSec < 3600) {
    const mins = Math.floor(diffSec / 60);
    return `${mins} minute${mins !== 1 ? "s" : ""} ago`;
  }
  if (diffSec < 86400) {
    const hours = Math.floor(diffSec / 3600);
    return `${hours} hour${hours !== 1 ? "s" : ""} ago`;
  }
  const days = Math.floor(diffSec / 86400);
  return `${days} day${days !== 1 ? "s" : ""} ago`;
};
