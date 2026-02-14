import type {
  FileSystemDirectoryEntry,
  FileSystemEntry,
  FileSystemFileEntry,
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types";
import { generateRandomString } from "@/lib/utils";

export const processFileEntry = (
  entry: FileSystemFileEntry,
  currentPath: string,
): Promise<FileWithPath | null> =>
  new Promise((resolve) => {
    entry.file((file) => {
      resolve({ file, relativePath: currentPath });
    });
  });

export const processEntry = async (
  entry: FileSystemEntry,
  currentPath = "",
): Promise<Array<FileWithPath>> => {
  if (entry.isFile) {
    const fileWithPath = await processFileEntry(
      entry as FileSystemFileEntry,
      currentPath,
    );
    return fileWithPath ? [fileWithPath] : [];
  }

  if (entry.isDirectory) {
    const dirReader = (entry as FileSystemDirectoryEntry).createReader();
    const allFiles: Array<FileWithPath> = [];
    const newPath = currentPath ? `${currentPath}/${entry.name}` : entry.name;

    return new Promise((resolve) => {
      const readEntries = () => {
        dirReader.readEntries(async (entries: Array<FileSystemEntry>) => {
          if (entries.length === 0) {
            resolve(allFiles);
            return;
          }

          for (const childEntry of entries) {
            const childFiles = await processEntry(childEntry, newPath);
            allFiles.push(...childFiles);
          }

          readEntries();
        });
      };

      readEntries();
    });
  }

  return [];
};

export const processDroppedItems = async (
  items: DataTransferItemList,
): Promise<Array<FileWithPath>> => {
  const allFiles: Array<FileWithPath> = [];

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    if (item.kind === "file") {
      const entry = item.webkitGetAsEntry();
      if (entry) {
        const files = await processEntry(entry as FileSystemEntry);
        allFiles.push(...files);
      }
    }
  }

  return allFiles;
};

export const extractFilesFromDrop = async (
  dataTransfer: DataTransfer,
): Promise<Array<FileWithPath>> => {
  if (dataTransfer.items.length > 0) {
    const files = await processDroppedItems(dataTransfer.items);
    if (files.length > 0) {
      return files;
    }
  }

  // Fallback for simple file drops
  if (dataTransfer.files.length > 0) {
    return Array.from(dataTransfer.files).map((file) => ({
      file,
      relativePath: "",
    }));
  }

  return [];
};

export const extractFolderPaths = (
  filesWithPaths: Array<FileWithPath>,
): Array<string> => {
  const folderPathsSet = new Set<string>();

  for (const { relativePath } of filesWithPaths) {
    if (relativePath) {
      const pathParts = relativePath.split("/");
      let currentPath = "";
      for (const part of pathParts) {
        currentPath = currentPath ? `${currentPath}/${part}` : part;
        folderPathsSet.add(currentPath);
      }
    }
  }

  return Array.from(folderPathsSet).sort(
    (a, b) => a.split("/").length - b.split("/").length,
  );
};

export const getFileExtension = (filename: string): string => {
  const dotIndex = filename.lastIndexOf(".");
  if (dotIndex === -1 || dotIndex === 0) return "";
  return filename.slice(dotIndex + 1).toLowerCase();
};

export const toStagedFiles = (files: Array<FileWithPath>): Array<StagedFile> =>
  files.map((f) => ({
    id: generateRandomString(12),
    file: f.file,
    relativePath: f.relativePath,
    extension: getFileExtension(f.file.name),
  }));
