import type { FC } from "react";
import type { IDataTableRowProps } from "@/components/common/components/DataTable/DataTableRow";
import type { BucketItem } from "@/types/bucket.ts";
import { DataTableRow } from "@/components/common/components/DataTable/DataTableRow";
import { useBucketItemDnd } from "@/components/bucket-view/components/FilesDndProvider/hooks/useBucketItemDnd";
import { cn } from "@/lib/utils";

interface IBucketItemRowProps extends IDataTableRowProps<BucketItem> {
  enabled: boolean;
  getSelectedIds: () => Array<string>;
}

export const BucketItemRow: FC<IBucketItemRowProps> = ({
  enabled,
  getSelectedIds,
  onPointerDown,
  className,
  style,
  ...props
}) => {
  const { row } = props;
  const { attributes, listeners, setNodeRef, isDragging, isOver } =
    useBucketItemDnd({
      item: row,
      enabled,
      selectedIds: getSelectedIds(),
    });

  return (
    <DataTableRow
      {...props}
      {...attributes}
      ref={setNodeRef}
      onPointerDown={(event) => {
        onPointerDown?.(event);
        listeners?.onPointerDown(event);
      }}
      style={isDragging ? { ...style, opacity: 0.4 } : style}
      className={cn(
        className,
        isOver && "bg-primary/10 outline-primary outline-2 -outline-offset-2",
      )}
    />
  );
};
