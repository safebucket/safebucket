import { FileDiff, FileDown, FileMinus, FileUp, Share2 } from "lucide-react";

import { ActivityMessage } from "@/components/activity-view/helpers/types";

export const messageMap = {
  BUCKET_CREATED: {
    message: "Shared a bucket '%%BUCKET_NAME%%' with you.",
    icon: Share2,
    iconColor: "text-green-500",
    iconBg: "bg-green-100",
  },
  FILE_UPLOADED: {
    message: "Uploaded a file '%%USERNAME%%' to the bucket '%%BUCKET_NAME%%'.",
    icon: FileUp,
    iconColor: "text-purple-500",
    iconBg: "bg-purple-100",
  },
  FILE_DOWNLOADED: {
    message: "Downloaded a file '%%NAME%%' from the bucket '%%BUCKET_NAME%%'.",
    icon: FileDown,
    iconColor: "text-indigo-500",
    iconBg: "bg-indigo-100",
  },
  FILE_UPDATED: {
    message: "Updated a file '%%USERNAME%%' from the bucket '%%BUCKET_NAME%%'.",
    icon: FileDiff,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-100",
  },
  FILE_DELETED: {
    message: "Deleted a file '%%USERNAME%%' from the bucket '%%BUCKET_NAME%%'.",
    icon: FileMinus,
    iconColor: "text-red-500",
    iconBg: "bg-red-100",
  },
} satisfies Record<ActivityMessage, object>;
