import { Database, File, Folder, Link2, Smartphone, User } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { formatAction } from "../helpers/format";
import { TimestampCell } from "./TimestampCell";
import type { TFunction } from "i18next";
import type { ColumnDef } from "@tanstack/react-table";
import type { ElementType } from "react";
import type { IActivity } from "@/types/activity";
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
    className: "text-chart-2 bg-chart-2/10",
  },
  file: {
    icon: File,
    className: "text-chart-1 bg-chart-1/10",
  },
  folder: {
    icon: Folder,
    className: "text-chart-3 bg-chart-3/10",
  },
  member: {
    icon: User,
    className: "text-chart-4 bg-chart-4/10",
  },
  user: {
    icon: User,
    className: "text-chart-4 bg-chart-4/10",
  },
  mfa_device: {
    icon: Smartphone,
    className: "text-chart-5 bg-chart-5/10",
  },
  share: {
    icon: Link2,
    className: "text-chart-2 bg-chart-2/10",
  },
};

const getResourceName = (activity: IActivity): string => {
  if (activity.file) {
    return activity.file.name;
  }
  if (activity.folder) {
    return activity.folder.name;
  }
  if (activity.bucket) {
    return activity.bucket.name;
  }
  if (activity.bucket_member_email) {
    return activity.bucket_member_email;
  }
  if (activity.share) {
    return activity.share.name;
  }
  if (activity.mfa_device) {
    return activity.mfa_device.name;
  }
  return "-";
};

const isResourceDeleted = (activity: IActivity): boolean => {
  return Boolean(activity.file?.deleted_at || activity.bucket?.deleted_at);
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

export const createColumns = (t: TFunction): Array<ColumnDef<IActivity>> => [
  {
    accessorKey: "timestamp",
    header: t("admin.activity.columns.timestamp"),
    cell: ({ row }) => <TimestampCell timestamp={row.original.timestamp} />,
  },
  {
    accessorKey: "user",
    header: t("admin.activity.columns.user"),
    cell: ({ row }) => {
      const user = row.original.user;
      return user?.email ?? t("activity.share_link");
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
