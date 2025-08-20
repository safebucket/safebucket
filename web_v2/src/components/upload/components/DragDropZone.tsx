import React, { type DragEvent, type FC, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";

import { cn } from "@/lib/utils";
import { Upload } from "lucide-react";
import { mutate } from "swr";

import { FileType } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { api_createFile } from "@/components/upload/helpers/api";
import type {
  FileSystemDirectoryEntry,
  FileSystemEntry,
  FileSystemFileEntry,
} from "@/components/upload/helpers/types";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

interface IDragDropZoneProps {
  bucketId: string;
  children: React.ReactNode;
  className?: string;
}

export const DragDropZone: FC<IDragDropZoneProps> = ({
  bucketId,
  children,
  className,
}) => {
  const { t } = useTranslation();
  const [isDragOver, setIsDragOver] = useState(false);
  const [_dragCounter, setDragCounter] = useState(0);

  const { startUpload } = useUploadContext();
  const { path } = useBucketViewContext();

  const handleDragEnter = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    setDragCounter((prev) => prev + 1);

    // Show overlay for any drag operation that includes files
    if (e.dataTransfer.types.includes("Files")) {
      setIsDragOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    setDragCounter((prev) => {
      const newCount = prev - 1;
      if (newCount <= 0) {
        setIsDragOver(false);
        return 0;
      }
      return newCount;
    });
  }, []);

  const handleDragOver = useCallback(
    (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      // Ensure overlay stays visible during drag over
      if (e.dataTransfer.types.includes("Files") && !isDragOver) {
        setIsDragOver(true);
      }
    },
    [isDragOver],
  );

  const processFileEntry = useCallback(
    (entry: FileSystemFileEntry, currentPath: string): Promise<Array<File>> => {
      return new Promise((resolve) => {
        entry.file((file) => {
          const fullPath = currentPath
            ? `${currentPath}/${file.name}`
            : file.name;
          const fileWithPath = new File([file], fullPath, { type: file.type });
          resolve([fileWithPath]);
        });
      });
    },
    [],
  );

  const processDirectoryEntry = useCallback(
    async (
      entry: FileSystemEntry,
      currentPath: string = "",
    ): Promise<Array<File>> => {
      return new Promise((resolve) => {
        if (entry.isFile) {
          processFileEntry(entry as FileSystemFileEntry, currentPath).then(
            resolve,
          );
          return;
        }

        if (entry.isDirectory) {
          const dirReader = (entry as FileSystemDirectoryEntry).createReader();
          const allFiles: Array<File> = [];

          const readEntries = () => {
            dirReader.readEntries(async (entries: Array<FileSystemEntry>) => {
              if (entries.length === 0) {
                resolve(allFiles);
                return;
              }

              for (const childEntry of entries) {
                const childPath = currentPath
                  ? `${currentPath}/${entry.name}`
                  : entry.name;
                const childFiles = await processDirectoryEntry(
                  childEntry,
                  childPath,
                );
                allFiles.push(...childFiles);
              }

              readEntries();
            });
          };

          readEntries();
        }
      });
    },
    [processFileEntry],
  );

  const extractFolderPaths = useCallback(
    (files: Array<File>): Set<string> => {
      const folderPaths = new Set<string>();

      files.forEach((file) => {
        const fileName = file.name;
        const lastSlashIndex = fileName.lastIndexOf("/");
        if (lastSlashIndex >= 0) {
          const relativePath = fileName.substring(0, lastSlashIndex);
          const pathParts = relativePath.split("/");

          let currentPath = "";
          pathParts.forEach((part) => {
            currentPath = currentPath ? `${currentPath}/${part}` : part;
            const fullFolderPath =
              path && path !== "/"
                ? `${path}/${currentPath}`
                : `/${currentPath}`;
            folderPaths.add(fullFolderPath);
          });
        }
      });

      return folderPaths;
    },
    [path],
  );

  const createFolders = useCallback(
    async (folderPaths: Set<string>) => {
      const createFolderPromises = Array.from(folderPaths).map((folderPath) => {
        const parts = folderPath.split("/");
        const folderName = parts[parts.length - 1];
        const parentPath = parts.slice(0, -1).join("/");
        // Ensure parentPath is never empty - backend requires non-empty path
        const apiParentPath = parentPath || "/";

        return api_createFile(
          folderName,
          FileType.folder,
          apiParentPath,
          bucketId,
        );
      });

      await Promise.all(createFolderPromises);
      await mutate(`/buckets/${bucketId}`);
    },
    [bucketId],
  );

  const uploadFiles = useCallback(
    (files: Array<File>) => {
      files.forEach((file) => {
        const fileName = file.name;
        const lastSlashIndex = fileName.lastIndexOf("/");
        const baseName =
          lastSlashIndex >= 0
            ? fileName.substring(lastSlashIndex + 1)
            : fileName;
        const relativePath =
          lastSlashIndex >= 0 ? fileName.substring(0, lastSlashIndex) : "";
        const fullPath = relativePath
          ? path && path !== "/"
            ? `${path}/${relativePath}`
            : `/${relativePath}`
          : path;

        const singleFileList = Object.assign([file], {
          length: 1,
          item: (index: number) => (index === 0 ? file : null),
        }) as FileList;

        Object.defineProperty(file, "name", {
          value: baseName,
          writable: false,
          configurable: true,
        });

        startUpload(singleFileList, fullPath, bucketId);
      });
    },
    [startUpload, path, bucketId],
  );

  const createFoldersAndUploadFiles = useCallback(
    async (files: Array<File>) => {
      try {
        const folderPaths = extractFolderPaths(files);
        await createFolders(folderPaths);
        uploadFiles(files);
      } catch (error) {
        console.error("Error creating folders:", error);
      }
    },
    [extractFolderPaths, createFolders, uploadFiles],
  );

  const processDroppedItems = useCallback(
    async (items: DataTransferItemList): Promise<Array<File>> => {
      const allFiles: Array<File> = [];

      for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.kind === "file") {
          const entry = item.webkitGetAsEntry();
          if (entry) {
            const files = await processDirectoryEntry(entry);
            allFiles.push(...files);
          }
        }
      }

      return allFiles;
    },
    [processDirectoryEntry],
  );

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      setIsDragOver(false);
      setDragCounter(0);

      const items = e.dataTransfer.items;
      if (items.length > 0) {
        const allFiles = await processDroppedItems(items);
        if (allFiles.length > 0) {
          await createFoldersAndUploadFiles(allFiles);
        }
        return;
      }

      const files = e.dataTransfer.files;
      if (files.length > 0) {
        startUpload(files, path, bucketId);
      }
    },
    [
      processDroppedItems,
      createFoldersAndUploadFiles,
      startUpload,
      path,
      bucketId,
    ],
  );

  return (
    <div
      className={cn("relative", className)}
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}

      {isDragOver && (
        <>
          {/* Full viewport purple drop zone overlay */}
          <div className="bg-primary/10 border-primary fixed inset-0 z-50 flex items-center justify-center border-4 border-dashed">
            <div className="text-primary flex flex-col items-center justify-center space-y-4">
              <div className="relative">
                <Upload className="h-20 w-20" />
              </div>
              <div className="text-center">
                <p className="text-xl font-semibold">
                  {t("upload.drag_drop.drop_files")}
                </p>
                <p className="text-primary/80 text-base">
                  {t("upload.drag_drop.drop_description")}
                </p>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
};
