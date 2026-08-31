import { useDndContext, useDraggable, useDroppable } from "@dnd-kit/core";
import type { BucketItem } from "@/types/bucket.ts";
import {
  canDropInto,
  resolveDragIds,
} from "@/components/bucket-view/helpers/dnd";
import { isFolder } from "@/components/bucket-view/helpers/utils";

interface IUseBucketItemDndOptions {
  item: BucketItem;
  enabled: boolean;
  selectedIds: Array<string>;
}

export function useBucketItemDnd({
  item,
  enabled,
  selectedIds,
}: IUseBucketItemDndOptions) {
  const { active } = useDndContext();
  const dragIds = active?.data.current?.dragIds as Array<string> | undefined;

  const draggable = useDraggable({
    id: item.id,
    data: { itemId: item.id, dragIds: resolveDragIds(selectedIds, item.id) },
    disabled: !enabled,
  });

  const droppable = useDroppable({
    id: `drop-${item.id}`,
    data: { dropFolderId: item.id },
    disabled: !isFolder(item) || !canDropInto(dragIds, item.id),
  });

  const setNodeRef = (node: HTMLElement | null) => {
    draggable.setNodeRef(node);
    droppable.setNodeRef(node);
  };

  return {
    attributes: draggable.attributes,
    listeners: draggable.listeners,
    setNodeRef,
    isDragging: draggable.isDragging,
    isOver: droppable.isOver,
  };
}
