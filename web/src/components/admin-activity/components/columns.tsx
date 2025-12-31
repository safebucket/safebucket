import { formatDistanceToNow } from "date-fns";
import { Database, File, Folder, User } from "lucide-react";
import { Link } from "@tanstack/react-router";
import type { TFunction } from "i18next";
import type { ColumnDef } from "@tanstack/react-table";
import type { ElementType } from "react";
import type { ActivityMessage, IActivity } from "@/types/activity";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface ResourceTypeConfig {
  icon: ElementType;
  className: string;
}

const resourceTypeConfig: Record<string, ResourceTypeConfig> = {
  bucket: {
    icon: Database,
    className: "text-blue-600 bg-blue-50 dark:text-blue-400 dark:bg-blue-950",
  },
  file: {
    icon: File,
    className:
      "text-emerald-600 bg-emerald-50 dark:text-emerald-400 dark:bg-emerald-950",
  },
  folder: {
    icon: Folder,
    className:
      "text-amber-600 bg-amber-50 dark:text-amber-400 dark:bg-amber-950",
  },
  member: {
    icon: User,
    className:
      "text-amber-600 bg-amber-50 dark:text-amber-400 dark:bg-amber-950",
  },
  user: {
    icon: User,
    className:
      "text-purple-600 bg-purple-50 dark:text-purple-400 dark:bg-purple-950",
  },
};

const getResourceName = (activity: IActivity): string => {
  if (activity.file) {
    return activity.file.name;
  }
  if (activity.bucket) {
    return activity.bucket.name;
  }
  if (activity.bucket_member_email) {
    return activity.bucket_member_email;
  }
  return "-";
};

const isResourceDeleted = (activity: IActivity): boolean => {
  return (
    activity.file?.deleted_at !== null || activity.bucket?.deleted_at !== null
  );
};

const getResourceLink = (activity: IActivity): string | null => {
  if (isResourceDeleted(activity)) {
    return null;
  }

  const bucketId = activity.bucket_id || activity.bucket?.id;

  if (activity.file && bucketId) {
    const folderId = activity.file.folder_id;
    return folderId
      ? `/buckets/${bucketId}/${folderId}`
      : `/buckets/${bucketId}`;
  }

  if (bucketId) {
    return `/buckets/${bucketId}`;
  }

  return null;
};

const formatAction = (message: ActivityMessage): string => {
  return message
    .replace(/_/g, " ")
    .toLowerCase()
    .replace(/^\w/, (c) => c.toUpperCase());
};

export const createColumns = (t: TFunction): Array<ColumnDef<IActivity>> => [
  {
    accessorKey: "timestamp",
    header: t("admin.activity.columns.timestamp"),
    cell: ({ row }) => {
      const timestamp = row.original.timestamp;
      if (!timestamp) return "-";
      const date = new Date(Number(timestamp) / 1000000);
      return formatDistanceToNow(date, { addSuffix: true });
    },
  },
  {
    accessorKey: "user",
    header: t("admin.activity.columns.user"),
    cell: ({ row }) => {
      const user = row.original.user;
      return user.email;
    },
  },
  {
    accessorKey: "message",
    header: t("admin.activity.columns.action"),
    cell: ({ row }) => {
      const message = row.original.message;
      return formatAction(message);
    },
  },
  {
    accessorKey: "object_type",
    header: t("admin.activity.columns.resource_type"),
    cell: ({ row }) => {
      const objectType = row.original.object_type;
      const isMemberAction = row.original.message.includes("MEMBER");
      const displayType = isMemberAction ? "member" : objectType;
      const config = resourceTypeConfig[displayType];
      const Icon = config.icon;
      return (
        <span
          className={`inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium ${config.className}`}
        >
          <Icon className="size-3.5" />
          {displayType}
        </span>
      );
    },
  },
  {
    id: "resource_name",
    header: t("admin.activity.columns.resource_name"),
    cell: ({ row }) => {
      const activity = row.original;
      const name = getResourceName(activity);
      const link = getResourceLink(activity);
      const deleted = isResourceDeleted(activity);

      if (link && name !== "-") {
        return (
          <Link
            to={link}
            className="text-primary underline-offset-4 hover:underline"
          >
            {name}
          </Link>
        );
      }

      if (deleted && name !== "-") {
        return (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="text-muted-foreground cursor-default">
                {name}
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {t("admin.activity.tooltip.object_deleted")}
            </TooltipContent>
          </Tooltip>
        );
      }

      return name;
    },
  },
];
