import type { ColumnDef } from "@tanstack/react-table";

import type { BucketItem } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView.tsx";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { formatDate, formatFileSize } from "@/lib/utils.ts";

export const createColumns = (
  t: (key: string) => string,
  onDownload: (file: IFile) => void,
): Array<ColumnDef<BucketItem>> => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.name")} />
    ),
    size: 350,
    cell: ({ row }) => {
      const item = row.original;
      const itemIsFolder = isFolder(item);
      return (
        <div className="flex items-center space-x-2 overflow-hidden max-w-[calc(100vw-8rem)] md:max-w-87.5">
          <FileIconView
            className="text-primary h-5 w-5 shrink-0"
            isFolder={itemIsFolder}
            extension={!itemIsFolder ? item.extension : undefined}
          />
          <p className="truncate">{row.getValue("name")}</p>
        </div>
      );
    },
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.size")} />
    ),
    cell: ({ row }) => {
      const item = row.original;
      return isFolder(item) ? "-" : formatFileSize(item.size);
    },
  },
  {
    id: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("share_consumer.type")} />
    ),
    cell: ({ row }) => {
      const item = row.original;
      const itemType = isFolder(item) ? "folder" : item.extension;
      return <Badge variant="secondary">{itemType}</Badge>;
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("share_consumer.uploaded_at")}
      />
    ),
    cell: ({ row }) => formatDate(row.getValue("created_at")),
  },
  {
    id: "actions",
    size: 80,
    cell: ({ row }) => {
      const item = row.original;
      if (isFolder(item)) return null;
      return (
        <Button
          variant="ghost"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            onDownload(item);
          }}
        >
          {t("common.download")}
        </Button>
      );
    },
  },
];
