import type { DragEvent, SyntheticEvent } from "react";

const stop = (e: SyntheticEvent) => e.stopPropagation();

const blockDrop = (e: DragEvent) => {
  e.stopPropagation();
  e.preventDefault();
  e.dataTransfer.dropEffect = "none";
};

export const blockEvents = {
  onClick: stop,
  onDoubleClick: stop,
  onPointerDown: stop,
  onContextMenu: stop,
  onDragEnter: stop,
  onDragLeave: stop,
  onDragOver: blockDrop,
  onDrop: blockDrop,
};
