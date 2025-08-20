import {
  FileDiff,
  FileDown,
  FileMinus,
  FileUp,
  Share2,
  UserMinus,
  UserPen,
  UserPlus,
} from "lucide-react";
import type { ActivityMessage } from "@/types/activity.ts";

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
    message:
      "Downloaded a file '%%FILE_NAME%%' on the bucket '%%BUCKET_NAME%%'.",
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
  BUCKET_MEMBER_CREATED: {
    message: "Invited %%BUCKET_MEMBER_EMAIL%% to the bucket '%%BUCKET_NAME%%'.",
    icon: UserPlus,
    iconColor: "text-blue-500",
    iconBg: "bg-blue-100",
  },
  BUCKET_MEMBER_UPDATED: {
    message:
      "Updated the group of %%BUCKET_MEMBER_EMAIL%% on the bucket '%%BUCKET_NAME%%'.",
    icon: UserPen,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-100",
  },
  BUCKET_MEMBER_DELETED: {
    message:
      "Removed %%BUCKET_MEMBER_EMAIL%% from the bucket '%%BUCKET_NAME%%'.",
    icon: UserMinus,
    iconColor: "text-red-500",
    iconBg: "bg-red-100",
  },
} satisfies Record<ActivityMessage, object>;
