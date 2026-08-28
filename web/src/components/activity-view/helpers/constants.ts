import {
  Clock,
  Eye,
  FileDiff,
  FileDown,
  FileMinus,
  FileUp,
  FolderMinus,
  FolderPen,
  FolderPlus,
  Link2,
  Link2Off,
  RotateCcw,
  Share2,
  Smartphone,
  SmartphoneCharging,
  SmartphoneNfc,
  Trash2,
  UserMinus,
  UserPen,
  UserPlus,
} from "lucide-react";
import type { ActivityMessage } from "@/types/activity.ts";

const success = {
  iconColor: "text-success-subtle-foreground",
  iconBg: "bg-success-subtle",
};
const info = {
  iconColor: "text-info-subtle-foreground",
  iconBg: "bg-info-subtle",
};
const warning = {
  iconColor: "text-warning-subtle-foreground",
  iconBg: "bg-warning-subtle",
};
const destructive = {
  iconColor: "text-destructive-subtle-foreground",
  iconBg: "bg-destructive-subtle",
};

export const messageMap = {
  BUCKET_CREATED: {
    messageKey: "activity.messages.bucket_created",
    icon: Share2,
    ...success,
  },
  BUCKET_DELETED: {
    messageKey: "activity.messages.bucket_deleted",
    icon: Share2,
    ...destructive,
  },
  FILE_UPLOADED: {
    messageKey: "activity.messages.file_uploaded",
    icon: FileUp,
    ...info,
  },
  FILE_DOWNLOADED: {
    messageKey: "activity.messages.file_downloaded",
    icon: FileDown,
    ...info,
  },
  FILE_UPDATED: {
    messageKey: "activity.messages.file_updated",
    icon: FileDiff,
    ...info,
  },
  FILE_DELETED: {
    messageKey: "activity.messages.file_deleted",
    icon: FileMinus,
    ...destructive,
  },
  FILE_EXPIRED: {
    messageKey: "activity.messages.file_expired",
    icon: Clock,
    ...warning,
  },
  FILE_TRASHED: {
    messageKey: "activity.messages.file_trashed",
    icon: Trash2,
    ...warning,
  },
  FILE_RESTORED: {
    messageKey: "activity.messages.file_restored",
    icon: RotateCcw,
    ...success,
  },
  FOLDER_CREATED: {
    messageKey: "activity.messages.folder_created",
    icon: FolderPlus,
    ...success,
  },
  FOLDER_UPDATED: {
    messageKey: "activity.messages.folder_updated",
    icon: FolderPen,
    ...info,
  },
  FOLDER_TRASHED: {
    messageKey: "activity.messages.folder_trashed",
    icon: Trash2,
    ...warning,
  },
  FOLDER_RESTORED: {
    messageKey: "activity.messages.folder_restored",
    icon: RotateCcw,
    ...success,
  },
  FOLDER_DELETED: {
    messageKey: "activity.messages.folder_deleted",
    icon: FolderMinus,
    ...destructive,
  },
  BUCKET_MEMBER_CREATED: {
    messageKey: "activity.messages.bucket_member_created",
    icon: UserPlus,
    ...success,
  },
  BUCKET_MEMBER_UPDATED: {
    messageKey: "activity.messages.bucket_member_updated",
    icon: UserPen,
    ...info,
  },
  BUCKET_MEMBER_DELETED: {
    messageKey: "activity.messages.bucket_member_deleted",
    icon: UserMinus,
    ...destructive,
  },
  MFA_DEVICE_ENROLLED: {
    messageKey: "activity.messages.mfa_device_enrolled",
    icon: SmartphoneCharging,
    ...success,
  },
  MFA_DEVICE_VERIFIED: {
    messageKey: "activity.messages.mfa_device_verified",
    icon: SmartphoneNfc,
    ...success,
  },
  MFA_DEVICE_UPDATED: {
    messageKey: "activity.messages.mfa_device_updated",
    icon: Smartphone,
    ...info,
  },
  MFA_DEVICE_REMOVED: {
    messageKey: "activity.messages.mfa_device_removed",
    icon: Smartphone,
    ...destructive,
  },
  SHARE_CREATED: {
    messageKey: "activity.messages.share_created",
    icon: Link2,
    ...success,
  },
  SHARE_DELETED: {
    messageKey: "activity.messages.share_deleted",
    icon: Link2Off,
    ...destructive,
  },
  SHARE_EXPIRED: {
    messageKey: "activity.messages.share_expired",
    icon: Clock,
    ...warning,
  },
  SHARE_MAX_VIEWS_REACHED: {
    messageKey: "activity.messages.share_max_views_reached",
    icon: Eye,
    ...warning,
  },
  SHARE_FILE_DOWNLOADED: {
    messageKey: "activity.messages.share_file_downloaded",
    icon: FileDown,
    ...info,
  },
  SHARE_FILE_UPLOADED: {
    messageKey: "activity.messages.share_file_uploaded",
    icon: FileUp,
    ...info,
  },
} satisfies Record<ActivityMessage, object>;
