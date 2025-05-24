import { FileDiff, FileDown, FileMinus, FileUp, Share2 } from "lucide-react";

import { ActivityMessage } from "@/components/common/types/activity";

export const messageMap = {
  BUCKET_CREATED: {
    message: "Created a bucket '%%BUCKET_NAME%%' and shared it with you.",
    icon: Share2,
    iconColor: "text-green-500",
    iconBg: "bg-green-100",
  },
  FILE_UPLOADED: {
    message: "Uploaded a file '%%FILE_NAME%%' on the bucket '%%BUCKET_NAME%%'.",
    icon: FileUp,
    iconColor: "text-purple-500",
    iconBg: "bg-purple-100",
  },
  FILE_DOWNLOADED: {
    message: "Downloaded a file '%%FILE_NAME%%' on the bucket '%%BUCKET_NAME%%'.",
    icon: FileDown,
    iconColor: "text-indigo-500",
    iconBg: "bg-indigo-100",
  },
  FILE_UPDATED: {
    message: "Updated a file '%%FILE_NAME%%' on the bucket '%%BUCKET_NAME%%'.",
    icon: FileDiff,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-100",
  },
  FILE_DELETED: {
    message: "Deleted a file '%%FILE_NAME%%' on the bucket '%%BUCKET_NAME%%'.",
    icon: FileMinus,
    iconColor: "text-red-500",
    iconBg: "bg-red-100",
  },
} satisfies Record<ActivityMessage, object>;
