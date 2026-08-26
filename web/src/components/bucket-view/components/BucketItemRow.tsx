import { useDndContext, useDraggable, useDroppable } from "@dnd-kit/core";
import type { FC } from "react";
import type { IDataTableRowProps } from "@/components/common/components/DataTable/DataTableRow";
import type { BucketItem } from "@/types/bucket.ts";
import { DataTableRow } from "@/components/common/components/DataTable/DataTableRow";
import {
  canDropInto,
  resolveDragIds,
} from "@/components/bucket-view/helpers/dnd";
import { isFolder } from "@/components/bucket-view/helpers/utils";

interface IBucketItemRowProps extends IDataTableRowProps<BucketItem> {
  enabled: boolean;
  getSelectedIds: () => Array<string>;
}

export const BucketItemRow: FC<IBucketItemRowProps> = ({
  enabled,
  getSelectedIds,
  ...props
}) => {
  const { row } = props;
  const { active } = useDndContext();
  const dragIds = active?.data.current?.dragIds as Array<string> | undefined;
  const itemIsFolder = isFolder(row);

  const {
    attributes,
    listeners,
    setNodeRef: setDragRef,
    isDragging,
  } = useDraggable({
    id: row.id,
    data: { itemId: row.id, dragIds: resolveDragIds(getSelectedIds(), row.id) },
    disabled: !enabled,
  });

  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `drop-${row.id}`,
    data: { dropFolderId: row.id },
    disabled: !itemIsFolder || !canDropInto(dragIds, row.id),
  });

  return (
    <DataTableRow
      {...props}
      trRef={(node) => {
        setDragRef(node);
        setDropRef(node);
      }}
      trProps={{
        ...attributes,
        ...listeners,
        style: isDragging ? { opacity: 0.4 } : undefined,
        className: isOver
          ? "bg-primary/10 outline-primary outline-2 -outline-offset-2"
          : undefined,
      }}
    />
  );
};
