import { useState } from "react";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  pointerWithin,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import type { ReactNode } from "react";
import type { DragEndEvent, DragStartEvent } from "@dnd-kit/core";
import type { BucketItem } from "@/types/bucket.ts";
import { resolveDragIds } from "@/components/bucket-view/helpers/dnd";
import { cursorToTopLeft } from "@/components/bucket-view/helpers/cursorToTopLeft";
import { useMoveItems } from "@/components/bucket-view/components/FilesDndProvider/hooks/useMoveItems";
import { DragPreview } from "@/components/bucket-view/components/FilesDndProvider/components/DragPreview";

interface IFilesDndProviderProps {
  bucketId: string;
  items: Array<BucketItem>;
  selectedIds: Array<string>;
  onMoveSuccess: () => void;
  children: ReactNode;
}

export function FilesDndProvider({
  bucketId,
  items,
  selectedIds,
  onMoveSuccess,
  children,
}: IFilesDndProviderProps) {
  const [activeItems, setActiveItems] = useState<Array<BucketItem>>([]);
  const moveMutation = useMoveItems(bucketId, { onSuccess: onMoveSuccess });

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
  );

  const clearActiveItems = () => {
    setActiveItems([]);
    document.body.classList.remove("files-dragging");
  };

  const handleDragStart = (event: DragStartEvent) => {
    if (moveMutation.isPending) return;
    const source = items.find((item) => item.id === event.active.id);
    if (!source) return;
    const ids = resolveDragIds(selectedIds, source.id);
    setActiveItems(items.filter((item) => ids.includes(item.id)));
    document.body.classList.add("files-dragging");
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const moving = activeItems;
    clearActiveItems();

    const data = event.over?.data.current as
      { dropFolderId?: string | null } | undefined;
    if (!moving.length || !data || data.dropFolderId === undefined) return;

    const targetFolderId = data.dropFolderId ?? undefined;
    const movable = moving.filter((item) => item.folder_id !== targetFolderId);
    if (movable.length > 0) {
      moveMutation.mutate({ items: movable, targetFolderId });
    }
  };

  return (
    <DndContext
      sensors={moveMutation.isPending ? [] : sensors}
      collisionDetection={pointerWithin}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      onDragCancel={clearActiveItems}
    >
      {children}
      <DragOverlay dropAnimation={null} modifiers={[cursorToTopLeft]}>
        <DragPreview items={activeItems} />
      </DragOverlay>
    </DndContext>
  );
}
