import React, { FC, useCallback, useState, DragEvent } from "react";
import { cn } from "@/lib/utils";
import { Upload } from "lucide-react";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileType } from "@/components/bucket-view/helpers/types";
import { api_createFile } from "@/components/upload/helpers/api";
import { mutate } from "swr";
import { 
  FileSystemEntry, 
  FileSystemFileEntry, 
  FileSystemDirectoryEntry
} from "@/components/upload/helpers/types";

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
  const [isDragOver, setIsDragOver] = useState(false);
  const [_dragCounter, setDragCounter] = useState(0);
  
  const { startUpload } = useUploadContext();
  const { path } = useBucketViewContext();

  const handleDragEnter = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    setDragCounter(prev => prev + 1);
    
    // Show overlay for any drag operation that includes files
    if (e.dataTransfer?.types && e.dataTransfer.types.includes("Files")) {
      setIsDragOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    setDragCounter(prev => {
      const newCount = prev - 1;
      if (newCount <= 0) {
        setIsDragOver(false);
        return 0;
      }
      return newCount;
    });
  }, []);

  const handleDragOver = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    // Ensure overlay stays visible during drag over
    if (e.dataTransfer?.types && e.dataTransfer.types.includes("Files") && !isDragOver) {
      setIsDragOver(true);
    }
  }, [isDragOver]);

  const processFileEntry = useCallback(
    (entry: FileSystemFileEntry, currentPath: string): Promise<File[]> => {
      return new Promise((resolve) => {
        entry.file((file) => {
          const fullPath = currentPath ? `${currentPath}/${file.name}` : file.name;
          const fileWithPath = new File([file], fullPath, { type: file.type });
          resolve([fileWithPath]);
        });
      });
    },
    []
  );

  const processDirectoryEntry = useCallback(
    async (entry: FileSystemEntry, currentPath: string = ""): Promise<File[]> => {
      return new Promise((resolve) => {
        if (entry.isFile) {
          processFileEntry(entry as FileSystemFileEntry, currentPath).then(resolve);
          return;
        }
        
        if (entry.isDirectory) {
          const dirReader = (entry as FileSystemDirectoryEntry).createReader();
          const allFiles: File[] = [];

          const readEntries = () => {
            dirReader.readEntries(async (entries: FileSystemEntry[]) => {
              if (entries.length === 0) {
                resolve(allFiles);
                return;
              }

              for (const childEntry of entries) {
                const childPath = currentPath ? `${currentPath}/${entry.name}` : entry.name;
                const childFiles = await processDirectoryEntry(childEntry, childPath);
                allFiles.push(...childFiles);
              }

              readEntries();
            });
          };

          readEntries();
        }
      });
    },
    [processFileEntry]
  );

  const extractFolderPaths = useCallback((files: File[]): Set<string> => {
    const folderPaths = new Set<string>();
    
    files.forEach(file => {
      const fileName = file.name;
      const lastSlashIndex = fileName.lastIndexOf("/");
      if (lastSlashIndex >= 0) {
        const relativePath = fileName.substring(0, lastSlashIndex);
        const pathParts = relativePath.split("/");
        
        let currentPath = "";
        pathParts.forEach(part => {
          currentPath = currentPath ? `${currentPath}/${part}` : part;
          const fullFolderPath = path && path !== "/" ? `${path}/${currentPath}` : `/${currentPath}`;
          folderPaths.add(fullFolderPath);
        });
      }
    });
    
    return folderPaths;
  }, [path]);

  const createFolders = useCallback(async (folderPaths: Set<string>) => {
    const createFolderPromises = Array.from(folderPaths).map(folderPath => {
      const parts = folderPath.split("/");
      const folderName = parts[parts.length - 1];
      const parentPath = parts.slice(0, -1).join("/");
      // Ensure parentPath is never empty - backend requires non-empty path
      const apiParentPath = parentPath || "/";
      
      return api_createFile(folderName, FileType.folder, apiParentPath, bucketId);
    });

    await Promise.all(createFolderPromises);
    await mutate(`/buckets/${bucketId}`);
  }, [bucketId]);

  const uploadFiles = useCallback((files: File[]) => {
    files.forEach(file => {
      const fileName = file.name;
      const lastSlashIndex = fileName.lastIndexOf("/");
      const baseName = lastSlashIndex >= 0 ? fileName.substring(lastSlashIndex + 1) : fileName;
      const relativePath = lastSlashIndex >= 0 ? fileName.substring(0, lastSlashIndex) : "";
      const fullPath = relativePath ? (path && path !== "/" ? `${path}/${relativePath}` : `/${relativePath}`) : path;
      
      const singleFileList = Object.assign([file], {
        length: 1,
        item: (index: number) => index === 0 ? file : null,
      }) as FileList;
      
      Object.defineProperty(file, "name", {
        value: baseName,
        writable: false,
        configurable: true
      });
      
      startUpload(singleFileList, fullPath, bucketId);
    });
  }, [startUpload, path, bucketId]);

  const createFoldersAndUploadFiles = useCallback(
    async (files: File[]) => {
      try {
        const folderPaths = extractFolderPaths(files);
        await createFolders(folderPaths);
        uploadFiles(files);
      } catch (error) {
        console.error("Error creating folders:", error);
      }
    },
    [extractFolderPaths, createFolders, uploadFiles]
  );

  const processDroppedItems = useCallback(
    async (items: DataTransferItemList): Promise<File[]> => {
      const allFiles: File[] = [];
      
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
    [processDirectoryEntry]
  );

  const handleDrop = useCallback(
    async (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      
      setIsDragOver(false);
      setDragCounter(0);

      const items = e.dataTransfer?.items;
      if (items && items.length > 0) {
        const allFiles = await processDroppedItems(items);
        if (allFiles.length > 0) {
          await createFoldersAndUploadFiles(allFiles);
        }
        return;
      }

      const files = e.dataTransfer?.files;
      if (files && files.length > 0) {
        startUpload(files, path, bucketId);
      }
    },
    [processDroppedItems, createFoldersAndUploadFiles, startUpload, path, bucketId]
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
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-primary/10 border-4 border-dashed border-primary">
            <div className="flex flex-col items-center justify-center space-y-4 text-primary">
              <div className="relative">
                <Upload className="h-20 w-20" />
              </div>
              <div className="text-center">
                <p className="text-xl font-semibold">Drop files or folders to upload</p>
                <p className="text-base text-primary/80">
                  Files and folders will be uploaded to the current directory
                </p>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
};