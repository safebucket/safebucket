import React, { FC } from "react";

import { ColumnDef } from "@tanstack/react-table";

import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { IFile } from "@/components/bucket-view/helpers/types";
import { getFileType } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { DataTableColumnHeader } from "@/components/common/components/DataTable/DataColumnHeader";
import { DataTable } from "@/components/common/components/DataTable/DataTable";
import { DataTableRowActions } from "@/components/common/components/DataTable/DataTableRowActions";
import { Badge } from "@/components/ui/badge";

export const columns: ColumnDef<IFile>[] = [
  {
    accessorKey: "name",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Name" />
    ),
    cell: ({ row }) => (
      <div className="flex w-[350px] items-center space-x-2">
        <FileIconView
          extension={row.getValue("type")}
          className="h-5 w-5 text-primary"
        />
        <p>{row.getValue("name")}</p>
      </div>
    ),
  },
  {
    accessorKey: "size",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Size" />
    ),
    cell: ({ row }) => <div className="">{row.getValue("size")}</div>,
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "type",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Type" />
    ),
    cell: ({ row }) => (
      <div className="">
        <Badge variant="secondary">{getFileType(row.getValue("type"))}</Badge>
      </div>
    ),
    filterFn: (row, id, value) => {
      return value.includes(row.getValue(id));
    },
  },
  {
    accessorKey: "modified",
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Uploaded At" />
    ),
    cell: ({ row }) => <div className="">{row.getValue("modified")}</div>,
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
}

export const BucketListView: FC<IBucketListViewProps> = ({
  files,
}: IBucketListViewProps) => {
  const { selected, setSelected, openFolder } = useBucketViewContext();

  return (
    <DataTable
      columns={columns}
      data={files}
      selected={selected}
      onRowClick={setSelected}
      onRowDoubleClick={openFolder}
    />
  );
};
