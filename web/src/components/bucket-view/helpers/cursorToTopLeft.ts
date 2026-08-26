import type { Modifier } from "@dnd-kit/core";

export const cursorToTopLeft: Modifier = ({
  activatorEvent,
  activeNodeRect,
  transform,
}) => {
  const event = activatorEvent as PointerEvent | null;
  if (!event || !activeNodeRect || typeof event.clientX !== "number") {
    return transform;
  }

  return {
    ...transform,
    x: transform.x + (event.clientX - activeNodeRect.left),
    y: transform.y + (event.clientY - activeNodeRect.top),
  };
};
