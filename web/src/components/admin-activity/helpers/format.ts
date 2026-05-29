import type { ActivityMessage } from "@/types/activity";

export const formatAction = (message: ActivityMessage): string => {
  return message
    .replace(/_/g, " ")
    .toLowerCase()
    .replace(/^\w/, (c) => c.toUpperCase());
};
