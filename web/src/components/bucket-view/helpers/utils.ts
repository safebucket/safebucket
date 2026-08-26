import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import type { BucketItem } from "@/types/bucket.ts";

export const isFolder = (item: BucketItem): item is IFolder => {
  return !("extension" in item);
};

export const isFile = (item: BucketItem): item is IFile => {
  return "extension" in item;
};

export const itemsToShow = (
  files: Array<IFile>,
  folders: Array<IFolder>,
  folderId: string | undefined,
): Array<BucketItem> => {
  const folderItems = folders.filter(
    (folder) =>
      (!folderId && !folder.folder_id) || folder.folder_id === folderId,
  );

  const fileItems = files.filter(
    (file) => (!folderId && !file.folder_id) || file.folder_id === folderId,
  );

  return [...folderItems, ...fileItems];
};

export const getFolderPathTrail = (
  folders: Array<IFolder>,
  currentFolderId: string | undefined,
): Array<IFolder> => {
  if (!currentFolderId) return [];

  const trail: Array<IFolder> = [];
  let nextId: string | undefined = currentFolderId;

  while (nextId) {
    const current = folders.find((folder) => folder.id === nextId);
    if (!current) break;
    trail.unshift(current);
    nextId = current.folder_id;
  }

  return trail;
};
