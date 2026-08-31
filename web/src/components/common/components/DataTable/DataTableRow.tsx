import { useTranslation } from "react-i18next";
import type { HTMLAttributes, Ref } from "react";
import type {
  IDataTableColumn,
  IDataTableSelection,
} from "@/components/common/components/DataTable/DataTable";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

export const DATA_TABLE_CELL_CLASS = "px-4 py-3 align-middle";

export const DATA_TABLE_RESPONSIVE_HIDDEN = {
  sm: "hidden sm:table-cell",
  md: "hidden md:table-cell",
  lg: "hidden lg:table-cell",
  xl: "hidden xl:table-cell",
} as const;

export interface IDataTableRowProps<TData> {
  rowId: string;
  row: TData;
  columns: Array<IDataTableColumn<TData>>;
  selection?: IDataTableSelection<TData>;
  isSelected: boolean;
  isActive: boolean;
  interactive: boolean;
  onRowClick?: (row: TData) => void;
  onRowDoubleClick?: (row: TData) => void;
  trRef?: Ref<HTMLTableRowElement>;
  trProps?: HTMLAttributes<HTMLTableRowElement> & {
    draggable?: boolean;
    style?: HTMLAttributes<HTMLTableRowElement>["style"];
  };
}

export function DataTableRow<TData>({
  rowId,
  row,
  columns,
  selection,
  isSelected,
  isActive,
  interactive,
  onRowClick,
  onRowDoubleClick,
  trRef,
  trProps,
}: IDataTableRowProps<TData>) {
  const { t } = useTranslation();

  return (
    <tr
      ref={trRef}
      onClick={onRowClick ? () => onRowClick(row) : undefined}
      onDoubleClick={onRowDoubleClick ? () => onRowDoubleClick(row) : undefined}
      {...trProps}
      className={cn(
        "border-border hover:bg-muted/50 border-t transition-colors",
        interactive && "cursor-pointer",
        isActive && "bg-accent/60",
        isSelected && "bg-primary/5",
        trProps?.className,
      )}
    >
      {selection ? (
        <td
          className={DATA_TABLE_CELL_CLASS}
          onClick={(e) => e.stopPropagation()}
        >
          <Checkbox
            aria-label={t("bucket.bulk_download.select_row")}
            checked={isSelected}
            disabled={!(selection.getIsSelectable?.(row) ?? true)}
            onCheckedChange={(value) =>
              selection.onRowSelectionChange((prev) => {
                const next = { ...prev };
                if (value === true) next[rowId] = true;
                else delete next[rowId];
                return next;
              })
            }
          />
        </td>
      ) : null}
      {columns.map((column) => (
        <td
          key={column.id}
          className={cn(
            DATA_TABLE_CELL_CLASS,
            column.responsive &&
              DATA_TABLE_RESPONSIVE_HIDDEN[column.responsive],
            column.cellClassName,
          )}
        >
          {column.cell(row)}
        </td>
      ))}
    </tr>
  );
}
