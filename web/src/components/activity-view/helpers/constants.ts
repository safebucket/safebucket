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
    messageKey: "activity.messages.bucket_created",
    icon: Share2,
    iconColor: "text-green-500",
    iconBg: "bg-green-100",
  },
  FILE_UPLOADED: {
    messageKey: "activity.messages.file_uploaded",
    icon: FileUp,
    iconColor: "text-purple-500",
    iconBg: "bg-purple-100",
  },
  FILE_DOWNLOADED: {
    messageKey: "activity.messages.file_downloaded",
    icon: FileDown,
    iconColor: "text-indigo-500",
    iconBg: "bg-indigo-100",
  },
  FILE_UPDATED: {
    messageKey: "activity.messages.file_updated",
    icon: FileDiff,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-100",
  },
  FILE_DELETED: {
    messageKey: "activity.messages.file_deleted",
    icon: FileMinus,
    iconColor: "text-red-500",
    iconBg: "bg-red-100",
  },
  BUCKET_MEMBER_CREATED: {
    messageKey: "activity.messages.bucket_member_created",
    icon: UserPlus,
    iconColor: "text-blue-500",
    iconBg: "bg-blue-100",
  },
  BUCKET_MEMBER_UPDATED: {
    messageKey: "activity.messages.bucket_member_updated",
    icon: UserPen,
    iconColor: "text-amber-500",
    iconBg: "bg-amber-100",
  },
  BUCKET_MEMBER_DELETED: {
    messageKey: "activity.messages.bucket_member_deleted",
    icon: UserMinus,
    iconColor: "text-red-500",
    iconBg: "bg-red-100",
  },
} satisfies Record<ActivityMessage, object>;
