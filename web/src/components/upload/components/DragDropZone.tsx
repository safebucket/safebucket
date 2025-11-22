import React, { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Upload } from "lucide-react";
import type { DragEvent, FC } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type {
  FileSystemDirectoryEntry,
  FileSystemEntry,
  FileSystemFileEntry,
} from "@/components/upload/helpers/types";
import { cn } from "@/lib/utils";
import type { IFolder } from "@/types/folder";

import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { createFolderMutationFn } from "@/components/upload/helpers/api";

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
  const queryClient = useQueryClient();
  const [isDragOver, setIsDragOver] = useState(false);
  const [_dragCounter, setDragCounter] = useState(0);

  const { startUpload } = useUploadContext();
  const { folderId } = useBucketViewContext();

  // Store files with their relative paths
  interface FileWithPath {
    file: File;
    relativePath: string;
  }

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

  // Process file entry and track its relative path
  const processFileEntry = useCallback(
    (
      entry: FileSystemFileEntry,
      currentPath: string,
    ): Promise<FileWithPath | null> => {
      return new Promise((resolve) => {
        entry.file((file) => {
          resolve({
            file,
            relativePath: currentPath,
          });
        });
      });
    },
    [],
  );

  // Process directory entry recursively, tracking paths
  const processDirectoryEntry = useCallback(
    async (
      entry: FileSystemEntry,
      currentPath: string = "",
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
        const newPath = currentPath
          ? `${currentPath}/${entry.name}`
          : entry.name;

        return new Promise((resolve) => {
          const readEntries = () => {
            dirReader.readEntries(async (entries: Array<FileSystemEntry>) => {
              if (entries.length === 0) {
                resolve(allFiles);
                return;
              }

              for (const childEntry of entries) {
                const childFiles = await processDirectoryEntry(
                  childEntry,
                  newPath,
                );
                allFiles.push(...childFiles);
              }

              readEntries();
            });
          };

          readEntries();
        });
      }

      return [];
    },
    [processFileEntry],
  );

  const processDroppedItems = useCallback(
    async (items: DataTransferItemList): Promise<Array<FileWithPath>> => {
      const allFiles: Array<FileWithPath> = [];

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

  // Extract unique folder paths from files and sort by depth
  const extractFolderPaths = useCallback(
    (filesWithPaths: Array<FileWithPath>): Array<string> => {
      const folderPathsSet = new Set<string>();

      filesWithPaths.forEach(({ relativePath }) => {
        if (relativePath) {
          const pathParts = relativePath.split("/");
          // Build all intermediate paths
          let currentPath = "";
          pathParts.forEach((part) => {
            currentPath = currentPath ? `${currentPath}/${part}` : part;
            folderPathsSet.add(currentPath);
          });
        }
      });

      // Sort by depth (parent folders first)
      return Array.from(folderPathsSet).sort(
        (a, b) => a.split("/").length - b.split("/").length,
      );
    },
    [],
  );

  // Create folders hierarchically and return path->folderId mapping
  const createFolders = useCallback(
    async (
      folderPaths: Array<string>,
      parentFolderId: string | null,
    ): Promise<Map<string, string | null>> => {
      const pathToIdMap = new Map<string, string | null>();
      // Map empty path to the current folder ID (or null for root)
      pathToIdMap.set("", parentFolderId);

      for (const folderPath of folderPaths) {
        const pathParts = folderPath.split("/");
        const folderName = pathParts[pathParts.length - 1];
        const parentPath = pathParts.slice(0, -1).join("/");

        // Get parent ID from map (null means root level)
        const parentId = pathToIdMap.get(parentPath) ?? null;

        try {
          const folder: IFolder = await createFolderMutationFn({
            name: folderName,
            folderId: parentId,
            bucketId,
          });
          pathToIdMap.set(folderPath, folder.id);
        } catch (error) {
          console.error(`Failed to create folder ${folderPath}:`, error);
          // If folder creation fails, stop the process
          throw error;
        }
      }

      // Invalidate queries to refresh bucket view
      await queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });

      return pathToIdMap;
    },
    [bucketId, queryClient],
  );

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();

      setIsDragOver(false);
      setDragCounter(0);

      const items = e.dataTransfer.items;

      if (items.length > 0) {
        try {
          // Process all dropped items with their paths
          const filesWithPaths = await processDroppedItems(items);

          if (filesWithPaths.length === 0) {
            return;
          }

          // Check if any files have folder structure
          const hasNestedFiles = filesWithPaths.some((f) => f.relativePath);

          if (hasNestedFiles) {
            // Extract and create folder structure
            const folderPaths = extractFolderPaths(filesWithPaths);
            const pathToIdMap = await createFolders(folderPaths, folderId);

            // Upload each file to its corresponding folder
            for (const { file, relativePath } of filesWithPaths) {
              // relativePath already contains the parent folder path
              const targetFolderId = pathToIdMap.get(relativePath) ?? null;

              const fileList = Object.assign([file], {
                length: 1,
                item: (index: number) => (index === 0 ? file : null),
              }) as FileList;

              startUpload(fileList, bucketId, targetFolderId);
            }
          } else {
            // No folder structure, upload directly to current folder
            for (const { file } of filesWithPaths) {
              const fileList = Object.assign([file], {
                length: 1,
                item: (index: number) => (index === 0 ? file : null),
              }) as FileList;
              startUpload(fileList, bucketId, folderId);
            }
          }
        } catch (error) {
          console.error("Failed to process dropped items:", error);
        }
        return;
      }

      // Fallback for simple file drops (no directory structure)
      const files = e.dataTransfer.files;
      if (files.length > 0) {
        startUpload(files, bucketId, folderId);
      }
    },
    [
      processDroppedItems,
      extractFolderPaths,
      createFolders,
      startUpload,
      folderId,
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
