import type { FC } from "react";
import { useTranslation } from "react-i18next";

import { formatDate, formatFileSize } from "@/lib/utils";
import type { ColumnDef } from "@tanstack/react-table";

import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import type { IFile } from "@/components/bucket-view/helpers/types";
import { FileType } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { DataTableRowActions } from "@/components/common/components/DataTable/DataTableRowActions";
import { Badge } from "@/components/ui/badge";
import { DragDropZone } from "@/components/upload/components/DragDropZone";

const createColumns = (t: (key: string) => string): ColumnDef<IFile>[] => [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("bucket.list_view.name")} />
    ),
    cell: ({ row }) => (
      <div className="flex w-[350px] items-center space-x-2">
        <FileIconView
          className="text-primary h-5 w-5"
          type={row.getValue("type")}
          extension={row.original.extension}
        />
        <p>{row.getValue("name")}</p>
      </div>
    ),
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("bucket.list_view.size")} />
    ),
    cell: ({ row }) =>
      row.getValue("type") === FileType.folder
        ? "-"
        : formatFileSize(row.getValue("size")),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("bucket.list_view.type")} />
    ),
    cell: ({ row }) => (
      <Badge variant="secondary">{row.getValue("type")}</Badge>
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "created_at",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title={t("bucket.list_view.uploaded_at")} />
    ),
    cell: ({ row }) => formatDate(row.getValue("created_at")),
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
  files: IFile[];
  bucketId: string;
}

export const BucketListView: FC<IBucketListViewProps> = ({
  files,
  bucketId,
}: IBucketListViewProps) => {
  const { t } = useTranslation();
  const { selected, setSelected, openFolder } = useBucketViewContext();
  const columns = createColumns(t);

  return (
    <DragDropZone bucketId={bucketId}>
      <DataTable
        columns={columns}
        data={files}
        selected={selected}
        onRowClick={setSelected}
        onRowDoubleClick={openFolder}
      />
    </DragDropZone>
  );
};
