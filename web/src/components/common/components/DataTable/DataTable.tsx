import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { ReactNode } from "react";
import type { OnChangeFn, RowSelectionState } from "@tanstack/react-table";
import { SortableColumnHeader } from "@/components/common/components/DataTable/SortableColumnHeader";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

type SortDir = "asc" | "desc";

export interface IDataTableColumn<TData> {
  id: string;
  header: string;
  cell: (row: TData) => ReactNode;
  sortAccessor?: (row: TData) => string | number;
  responsive?: "sm" | "md" | "lg" | "xl";
  headerClassName?: string;
  cellClassName?: string;
}

export interface IDataTableSelection<TData> {
  rowSelection: RowSelectionState;
  onRowSelectionChange: OnChangeFn<RowSelectionState>;
  getIsSelectable?: (row: TData) => boolean;
}

interface IDataTableProps<TData> {
  columns: Array<IDataTableColumn<TData>>;
  data: Array<TData>;
  getRowId: (row: TData) => string;
  emptyMessage?: string;
  groupOrder?: (row: TData) => number;
  defaultSortId?: string;
  defaultSortDir?: SortDir;
  selection?: IDataTableSelection<TData>;
  onRowClick?: (row: TData) => void;
  onRowDoubleClick?: (row: TData) => void;
  isRowActive?: (row: TData) => boolean;
  isLoading?: boolean;
  skeletonRowCount?: number;
}

const cell = "px-4 py-3 align-middle";
const headCell = "px-4 py-2.5 text-left";

const RESPONSIVE_HIDDEN = {
  sm: "hidden sm:table-cell",
  md: "hidden md:table-cell",
  lg: "hidden lg:table-cell",
  xl: "hidden xl:table-cell",
} as const;

export function DataTable<TData>({
  columns,
  data,
  getRowId,
  emptyMessage = "",
  groupOrder,
  defaultSortId,
  defaultSortDir = "asc",
  selection,
  onRowClick,
  onRowDoubleClick,
  isRowActive,
  isLoading = false,
  skeletonRowCount = 3,
}: IDataTableProps<TData>) {
  const { t } = useTranslation();
  const [sortId, setSortId] = useState<string | null>(defaultSortId ?? null);
  const [sortDir, setSortDir] = useState<SortDir>(defaultSortDir);

  const columnCount = columns.length + (selection ? 1 : 0);
  const interactive = Boolean(onRowClick || onRowDoubleClick);

  const sorted = useMemo(() => {
    const sortColumn = columns.find(
      (column) => column.id === sortId && column.sortAccessor,
    );
    if (!groupOrder && !sortColumn) {
      return data;
    }
    const factor = sortDir === "asc" ? 1 : -1;
    return [...data].sort((a, b) => {
      if (groupOrder) {
        const groupDiff = groupOrder(a) - groupOrder(b);
        if (groupDiff !== 0) return groupDiff;
      }
      if (!sortColumn?.sortAccessor) return 0;
      const va = sortColumn.sortAccessor(a);
      const vb = sortColumn.sortAccessor(b);
      const cmp =
        typeof va === "number" && typeof vb === "number"
          ? va - vb
          : String(va).localeCompare(String(vb));
      return cmp * factor;
    });
  }, [columns, data, groupOrder, sortId, sortDir]);

  const selectableIds = useMemo(() => {
    if (!selection) return [];
    return sorted
      .filter((row) => selection.getIsSelectable?.(row) ?? true)
      .map(getRowId);
  }, [selection, sorted, getRowId]);

  const selectedCount = selectableIds.filter(
    (id) => selection?.rowSelection[id],
  ).length;
  const allSelected =
    selectableIds.length > 0 && selectedCount === selectableIds.length;
  const someSelected = selectedCount > 0 && !allSelected;

  const toggleSort = (id: string) => {
    if (sortId === id) {
      setSortDir((dir) => (dir === "asc" ? "desc" : "asc"));
    } else {
      setSortId(id);
      setSortDir("asc");
    }
  };

  const toggleAll = (value: boolean) => {
    selection?.onRowSelectionChange(() => {
      const next: RowSelectionState = {};
      if (value) selectableIds.forEach((id) => (next[id] = true));
      return next;
    });
  };

  const toggleRow = (id: string, value: boolean) => {
    selection?.onRowSelectionChange((prev) => {
      const next = { ...prev };
      if (value) next[id] = true;
      else delete next[id];
      return next;
    });
  };

  return (
    <div className="border-border bg-card overflow-hidden rounded-xl border">
      <table className="w-full border-collapse text-sm">
        <thead className="bg-muted">
          <tr>
            {selection ? (
              <th className={cn(headCell, "w-12")}>
                <Checkbox
                  aria-label={t("bucket.bulk_download.select_all")}
                  checked={allSelected || (someSelected && "indeterminate")}
                  onCheckedChange={(value) => toggleAll(value === true)}
                />
              </th>
            ) : null}
            {columns.map((column) => (
              <th
                key={column.id}
                className={cn(
                  headCell,
                  column.responsive && RESPONSIVE_HIDDEN[column.responsive],
                  column.headerClassName,
                )}
              >
                {column.sortAccessor ? (
                  <SortableColumnHeader
                    label={column.header}
                    active={sortId === column.id}
                    dir={sortDir}
                    onClick={() => toggleSort(column.id)}
                  />
                ) : column.header ? (
                  <span className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                    {column.header}
                  </span>
                ) : null}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {isLoading ? (
            Array.from({ length: skeletonRowCount }, (_, index) => (
              <tr key={index} className="border-border border-t">
                <td className={cell} colSpan={columnCount}>
                  <Skeleton className="h-6 w-full" />
                </td>
              </tr>
            ))
          ) : sorted.length === 0 ? (
            <tr>
              <td
                colSpan={columnCount}
                className="text-muted-foreground h-24 text-center"
              >
                {emptyMessage}
              </td>
            </tr>
          ) : (
            sorted.map((row) => {
              const id = getRowId(row);
              const isSelected = Boolean(selection?.rowSelection[id]);
              return (
                <tr
                  key={id}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                  onDoubleClick={
                    onRowDoubleClick ? () => onRowDoubleClick(row) : undefined
                  }
                  className={cn(
                    "border-border hover:bg-muted/50 border-t transition-colors",
                    interactive && "cursor-pointer",
                    isRowActive?.(row) && "bg-accent/60",
                    isSelected && "bg-primary/5",
                  )}
                >
                  {selection ? (
                    <td className={cell} onClick={(e) => e.stopPropagation()}>
                      <Checkbox
                        aria-label={t("bucket.bulk_download.select_row")}
                        checked={isSelected}
                        disabled={!(selection.getIsSelectable?.(row) ?? true)}
                        onCheckedChange={(value) =>
                          toggleRow(id, value === true)
                        }
                      />
                    </td>
                  ) : null}
                  {columns.map((column) => (
                    <td
                      key={column.id}
                      className={cn(
                        cell,
                        column.responsive &&
                          RESPONSIVE_HIDDEN[column.responsive],
                        column.cellClassName,
                      )}
                    >
                      {column.cell(row)}
                    </td>
                  ))}
                </tr>
              );
            })
          )}
        </tbody>
      </table>
    </div>
  );
}
