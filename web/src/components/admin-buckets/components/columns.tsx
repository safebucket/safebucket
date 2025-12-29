import { formatDistanceToNow } from "date-fns";
import { Link } from "@tanstack/react-router";
import type { TFunction } from "i18next";
import type { ColumnDef } from "@tanstack/react-table";
import type { IAdminBucket } from "@/queries/admin";
import { formatFileSize } from "@/lib/utils";

export const createColumns = (t: TFunction): Array<ColumnDef<IAdminBucket>> => [
  {
    accessorKey: "name",
    header: t("admin.buckets.columns.name"),
    cell: ({ row }) => {
      const bucket = row.original;
      return (
        <Link
          to="/buckets/$bucketId/{-$folderId}"
          params={{ bucketId: bucket.id, folderId: undefined }}
          className="font-medium text-primary hover:underline"
        >
          {bucket.name}
        </Link>
      );
    },
  },
  {
    accessorKey: "creator.email",
    header: t("admin.buckets.columns.creator"),
    cell: ({ row }) => {
      const creator = row.original.creator;
      return (
        <div>
          <div className="font-medium">{creator.email}</div>
          <div className="text-xs text-muted-foreground">
            {creator.first_name} {creator.last_name}
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "member_count",
    header: t("admin.buckets.columns.members"),
  },
  {
    accessorKey: "file_count",
    header: t("admin.buckets.columns.files"),
  },
  {
    accessorKey: "size",
    header: t("admin.buckets.columns.size"),
    cell: ({ row }) => formatFileSize(row.getValue("size")),
  },
  {
    accessorKey: "created_at",
    header: t("admin.buckets.columns.created"),
    cell: ({ row }) => {
      return formatDistanceToNow(new Date(row.getValue("created_at")), {
        addSuffix: true,
      });
    },
  },
];
