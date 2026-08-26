export const resolveDragIds = (
  selectedIds: Array<string>,
  itemId: string,
): Array<string> =>
  selectedIds.includes(itemId) && selectedIds.length > 1
    ? [...selectedIds]
    : [itemId];

export const canDropInto = (
  dragIds: Array<string> | undefined,
  folderId: string | undefined,
): boolean => {
  if (!dragIds || folderId === undefined) return true;
  return !dragIds.includes(folderId);
};
