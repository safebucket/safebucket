import React from "react";

import {
  flexRender,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
} from "@tanstack/react-table";
import { useTranslation } from "react-i18next";
import type {
  ColumnDef,
  ColumnFiltersState,
  OnChangeFn,
  Row,
  RowSelectionState,
  SortingState,
  VisibilityState,
} from "@tanstack/react-table";

import { FileActions } from "@/components/file-actions/FileActions";
import { TrashActions } from "@/components/file-actions/components/TrashActions";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

interface ColumnMeta {
  className?: string;
}

interface DataTableProps<TData, TValue> {
  columns: Array<ColumnDef<TData, TValue>>;
  data: Array<TData>;
  selected: TData | null;
  onRowClick: (row: TData) => void;
  onRowDoubleClick: (row: TData) => void;
  trashMode?: boolean;
  onRestore?: (fileId: string, fileName: string) => void;
  onPermanentDelete?: (fileId: string, fileName: string) => void;
  defaultColumnVisibility?: VisibilityState;
  rowSelection?: RowSelectionState;
  onRowSelectionChange?: OnChangeFn<RowSelectionState>;
  getRowId?: (row: TData) => string;
  enableRowSelection?: boolean | ((row: Row<TData>) => boolean);
}

export function DataTable<TData extends { id: string; name: string }, TValue>({
  columns,
  data,
  selected,
  onRowClick,
  onRowDoubleClick,
  trashMode = false,
  onRestore,
  onPermanentDelete,
  defaultColumnVisibility,
  rowSelection,
  onRowSelectionChange,
  getRowId,
  enableRowSelection = true,
}: DataTableProps<TData, TValue>) {
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>(defaultColumnVisibility ?? {});

  React.useEffect(() => {
    if (defaultColumnVisibility) {
      setColumnVisibility(defaultColumnVisibility);
    }
  }, [defaultColumnVisibility]);
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
      rowSelection: rowSelection ?? {},
    },
    enableRowSelection,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: onRowSelectionChange,
    getRowId,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
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
                  const meta = header.column.columnDef.meta as
                    | ColumnMeta
                    | undefined;
                  return (
                    <TableHead
                      key={header.id}
                      colSpan={header.colSpan}
                      className={cn("px-4", meta?.className)}
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
              table.getRowModel().rows.map((row) => {
                const tableRow = (
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
                    {row.getVisibleCells().map((cell) => {
                      const meta = cell.column.columnDef.meta as
                        | ColumnMeta
                        | undefined;
                      return (
                        <TableCell
                          key={cell.id}
                          className={cn("select-none", meta?.className)}
                        >
                          {flexRender(
                            cell.column.columnDef.cell,
                            cell.getContext(),
                          )}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                );

                return trashMode ? (
                  <TrashActions
                    key={row.id}
                    file={row.original as any}
                    type="context"
                    onRestore={onRestore}
                    onPermanentDelete={onPermanentDelete}
                  >
                    {tableRow}
                  </TrashActions>
                ) : (
                  <FileActions
                    key={row.id}
                    file={row.original as any}
                    type="context"
                  >
                    {tableRow}
                  </FileActions>
                );
              })
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
    </div>
  );
}
