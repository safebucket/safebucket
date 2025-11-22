import { useTranslation } from "react-i18next";
import { CheckCircle, LoaderCircle, Trash2 } from "lucide-react";
import type { FC } from "react";

import type { ColumnDef } from "@tanstack/react-table";

import { FileStatus } from "@/types/file.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { formatDate, formatFileSize } from "@/lib/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { DataTableRowActions } from "@/components/common/components/DataTable/DataTableRowActions";
import { Badge } from "@/components/ui/badge";
import { DragDropZone } from "@/components/upload/components/DragDropZone";
import type { BucketItem } from "@/types/bucket.ts";

const createColumns = (
  t: (key: string) => string,
): Array<ColumnDef<BucketItem>> => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.list_view.name")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      const itemIsFolder = isFolder(item);
      return (
        <div className="flex w-[350px] items-center space-x-2">
          <FileIconView
            className="text-primary h-5 w-5"
            isFolder={itemIsFolder}
            extension={!itemIsFolder ? item.extension : undefined}
          />
          <p>{row.getValue("name")}</p>
        </div>
      );
    },
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.list_view.size")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      return isFolder(item) ? "-" : formatFileSize(item.size);
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: "type",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.list_view.type")}
      />
    ),
    cell: ({ row }) => {
      const item = row.original;
      const itemType = isFolder(item) ? "folder" : item.extension;
      return <Badge variant="secondary">{itemType}</Badge>;
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.list_view.uploaded_at")}
      />
    ),
    cell: ({ row }) => formatDate(row.getValue("created_at")),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "status",
    header: ({ column }) => (
      <DataTableColumnHeader
        column={column}
        title={t("bucket.list_view.status")}
      />
    ),
    cell: ({ row }) => {
      const status = row.getValue("status");

      switch (status) {
        case FileStatus.uploaded:
          return (
            <Badge className="bg-green-100 text-green-800 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800">
              <CheckCircle className="h-3 w-3" />
              {t("bucket.list_view.uploaded")}
            </Badge>
          );
        case FileStatus.uploading:
          return (
            <Badge className="bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              {t("bucket.list_view.uploading")}
            </Badge>
          );
        case FileStatus.deleting:
          return (
            <Badge className="bg-red-100 text-red-800 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              {t("bucket.list_view.deleting")}
            </Badge>
          );
        case FileStatus.trashed:
          return (
            <Badge className="bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/20 dark:text-orange-300 dark:border-orange-800">
              <Trash2 className="h-3 w-3" />
              {t("bucket.trash_view.trashed")}
            </Badge>
          );
        case FileStatus.restoring:
          return (
            <Badge className="bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/20 dark:text-blue-300 dark:border-blue-800">
              <LoaderCircle className="h-3 w-3 animate-spin" />
              {t("bucket.trash_view.restoring")}
            </Badge>
          );
        default:
          return "-";
      }
    },
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    id: "actions",
    cell: ({ row }) => <DataTableRowActions row={row} />,
  },
];

interface IBucketListViewProps {
  items: Array<BucketItem>;
  bucketId: string;
}

export const BucketListView: FC<IBucketListViewProps> = ({
  items,
  bucketId,
}: IBucketListViewProps) => {
  const { t } = useTranslation();
  const { selected, setSelected, openFolder } = useBucketViewContext();
  const columns = createColumns(t);

  return (
    <DragDropZone bucketId={bucketId}>
      <DataTable
        columns={columns}
        data={items}
        selected={selected}
        onRowClick={setSelected}
        onRowDoubleClick={openFolder}
      />
    </DragDropZone>
  );
};
