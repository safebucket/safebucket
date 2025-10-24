import React from "react";

import {
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useTranslation } from "react-i18next";
import type {
  ColumnDef,
  ColumnFiltersState,
  SortingState,
  VisibilityState,
} from "@tanstack/react-table";

import type { IFile } from "@/types/file.ts";
import { FileActions } from "@/components/FileActions/FileActions";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface DataTableProps<TData, TValue> {
  columns: Array<ColumnDef<TData, TValue>>;
  data: Array<TData>;
  selected: TData | null;
  onRowClick: (row: TData) => void;
  onRowDoubleClick: (row: TData) => void;
  trashMode?: boolean;
  onRestore?: (fileId: string, fileName: string) => void;
  onPermanentDelete?: (fileId: string, fileName: string) => void;
}

export function DataTable<TData extends IFile, TValue>({
  columns,
  data,
  selected,
  onRowClick,
  onRowDoubleClick,
  trashMode = false,
  onRestore,
  onPermanentDelete,
}: DataTableProps<TData, TValue>) {
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>(
    [],
  );
  const [sorting, setSorting] = React.useState<SortingState>([]);

  const table = useReactTable({
    data,
    columns,
    state: {
      sorting,
      columnVisibility,
      columnFilters,
    },
    enableRowSelection: true,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
  });

  const { t } = useTranslation();

  return (
    <div className="space-y-4">
      <div className="cursor bg-primary-foreground rounded-md border shadow-sm">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className="bg-muted/50">
                {headerGroup.headers.map((header) => {
                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      className="px-4"
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(
                            header.column.columnDef.header,
                            header.getContext(),
                          )}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <FileActions
                  key={row.id}
                  file={row.original}
                  type="context"
                  trashMode={trashMode}
                  onRestore={onRestore}
                  onPermanentDelete={onPermanentDelete}
                >
                  <TableRow
                    key={row.id}
                    data-state={
                      selected && selected.id === row.original.id && "selected"
                    }
                    className="cursor-pointer"
                    onClick={() => onRowClick(row.original)}
                    onDoubleClick={() => onRowDoubleClick(row.original)}
                    onContextMenu={() => onRowClick(row.original)}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id} className="select-none">
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext(),
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                </FileActions>
              ))
            ) : (
              <TableRow>
                <TableCell
                  colSpan={columns.length}
                  className="h-24 text-center"
                >
                  {t("bucket.list_view.empty_folder")}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {/* <DataTablePagination table={table} />*/}
    </div>
  );
}
