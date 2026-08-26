import { useDndContext, useDroppable } from "@dnd-kit/core";
import type { ReactNode } from "react";
import { canDropInto } from "@/components/bucket-view/helpers/dnd";
import { cn } from "@/lib/utils";

interface IDroppableAreaProps {
  folderId: string | undefined;
  disabled?: boolean;
  children: ReactNode;
}

export function DroppableArea({
  folderId,
  disabled = false,
  children,
}: IDroppableAreaProps) {
  const { active } = useDndContext();
  const dragIds = active?.data.current?.dragIds as Array<string> | undefined;

  const { setNodeRef, isOver } = useDroppable({
    id: `drop-zone-${folderId ?? "root"}`,
    data: { dropFolderId: folderId ?? null },
    disabled: disabled || !canDropInto(dragIds, folderId),
  });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "rounded",
        isOver && "bg-primary/10 ring-primary/60 text-primary ring-2",
      )}
    >
      {children}
    </div>
  );
}
