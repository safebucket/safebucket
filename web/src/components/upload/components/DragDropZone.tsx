import React, { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { Upload } from "lucide-react";
import type { DragEvent, FC } from "react";

import type {
  FileSystemDirectoryEntry,
  FileSystemEntry,
  FileSystemFileEntry,
} from "@/components/upload/helpers/types";
import { cn } from "@/lib/utils";

import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
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
    const {folderId} = useBucketViewContext();

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

    // Simplified: Just process files without complex folder structure
  const processFileEntry = useCallback(
      (entry: FileSystemFileEntry): Promise<File | null> => {
      return new Promise((resolve) => {
          entry.file((file) => resolve(file));
      });
    },
    [],
  );

  const processDirectoryEntry = useCallback(
      async (entry: FileSystemEntry): Promise<Array<File>> => {
          if (entry.isFile) {
              const file = await processFileEntry(entry as FileSystemFileEntry);
              return file ? [file] : [];
          }

          if (entry.isDirectory) {
              const dirReader = (entry as FileSystemDirectoryEntry).createReader();
              const allFiles: Array<File> = [];

              return new Promise((resolve) => {
          const readEntries = () => {
            dirReader.readEntries(async (entries: Array<FileSystemEntry>) => {
              if (entries.length === 0) {
                resolve(allFiles);
                return;
              }

              for (const childEntry of entries) {
                  const childFiles = await processDirectoryEntry(childEntry);
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
        const currentFolderId = folderId ?? undefined;

      if (items.length > 0) {
        const allFiles = await processDroppedItems(items);
          // Upload all files from dropped folders to current folder
          for (const file of allFiles) {
              const fileList = Object.assign([file], {
                  length: 1,
                  item: (index: number) => (index === 0 ? file : null),
              }) as FileList;
              startUpload(fileList, currentFolderId, bucketId);
        }
        return;
      }

      const files = e.dataTransfer.files;
      if (files.length > 0) {
          startUpload(files, currentFolderId, bucketId);
      }
    },
      [processDroppedItems, startUpload, folderId, bucketId],
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
