import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";

// Union type for items that can be displayed in the bucket view
export type BucketItem = IFile | IFolder;

// Type guards to distinguish between files and folders
export const isFolder = (item: BucketItem): item is IFolder => {
  return !("extension" in item);
};

export const isFile = (item: BucketItem): item is IFile => {
  return "extension" in item;
};

// Get all items (files + folders) for a specific folder
export const itemsToShow = (
  files: Array<IFile>,
  folders: Array<IFolder>,
  folderId: string | null,
): Array<BucketItem> => {
  const folderItems = folders.filter(
    (folder) =>
      (folderId === null && !folder.folder_id) || folder.folder_id === folderId,
  );

  const fileItems = files.filter(
    (file) =>
      (folderId === null && !file.folder_id) || file.folder_id === folderId,
  );

  return [...folderItems, ...fileItems];
};
