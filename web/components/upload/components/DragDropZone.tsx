import React, { FC, useCallback, useState, DragEvent } from "react";

import { cn } from "@/lib/utils";
import { Upload, FileIcon } from "lucide-react";

import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileType } from "@/components/bucket-view/helpers/types";
import { api_createFile } from "@/components/upload/helpers/api";
import { mutate } from "swr";

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
  const [dragCounter, setDragCounter] = useState(0);
  
  const { startUpload } = useUploadContext();
  const { path } = useBucketViewContext();

  const handleDragEnter = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    setDragCounter(prev => prev + 1);
    
    if (e.dataTransfer?.items && e.dataTransfer.items.length > 0) {
      setIsDragOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    setDragCounter(prev => prev - 1);
    
    if (dragCounter <= 1) {
      setIsDragOver(false);
    }
  }, [dragCounter]);

  const handleDragOver = useCallback((e: DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const processDirectoryEntry = useCallback(
    async (entry: FileSystemEntry, currentPath: string = ""): Promise<File[]> => {
      return new Promise((resolve) => {
        if (entry.isFile) {
          (entry as FileSystemFileEntry).file((file) => {
            // Create a new File object with the full path as the name
            const fullPath = currentPath ? `${currentPath}/${file.name}` : file.name;
            const fileWithPath = new File([file], fullPath, { type: file.type });
            resolve([fileWithPath]);
          });
        } else if (entry.isDirectory) {
          const dirReader = (entry as FileSystemDirectoryEntry).createReader();
          const allFiles: File[] = [];
          
          const readEntries = () => {
            dirReader.readEntries(async (entries) => {
              if (entries.length === 0) {
                resolve(allFiles);
                return;
              }
              
              for (const childEntry of entries) {
                const childPath = currentPath ? `${currentPath}/${entry.name}` : entry.name;
                const childFiles = await processDirectoryEntry(childEntry, childPath);
                allFiles.push(...childFiles);
              }
              
              readEntries(); // Continue reading if there are more entries
            });
          };
          
          readEntries();
        }
      });
    },
    []
  );

  const createFoldersAndUploadFiles = useCallback(
    async (files: File[]) => {
      // Extract unique folder paths that need to be created
      const folderPaths = new Set<string>();
      
      files.forEach(file => {
        const fileName = file.name;
        const lastSlashIndex = fileName.lastIndexOf('/');
        if (lastSlashIndex >= 0) {
          const relativePath = fileName.substring(0, lastSlashIndex);
          const pathParts = relativePath.split('/');
          
          // Add all parent folder paths
          let currentPath = '';
          pathParts.forEach(part => {
            currentPath = currentPath ? `${currentPath}/${part}` : part;
            const fullFolderPath = path ? `${path}/${currentPath}` : currentPath;
            folderPaths.add(fullFolderPath);
          });
        }
      });

      // Create folders first
      const createFolderPromises = Array.from(folderPaths).map(folderPath => {
        const parts = folderPath.split('/');
        const folderName = parts[parts.length - 1];
        const parentPath = parts.slice(0, -1).join('/');
        
        return api_createFile(folderName, FileType.folder, parentPath, bucketId);
      });

      try {
        await Promise.all(createFolderPromises);
        await mutate(`/buckets/${bucketId}`);
        
        // Upload files after folders are created
        for (const file of files) {
          const fileName = file.name;
          const lastSlashIndex = fileName.lastIndexOf('/');
          const baseName = lastSlashIndex >= 0 ? fileName.substring(lastSlashIndex + 1) : fileName;
          const relativePath = lastSlashIndex >= 0 ? fileName.substring(0, lastSlashIndex) : '';
          const fullPath = relativePath ? (path ? `${path}/${relativePath}` : relativePath) : path;
          
          // Create FileList with single file for startUpload
          const singleFileList = Object.assign([file], {
            length: 1,
            item: (index: number) => index === 0 ? file : null,
          }) as FileList;
          
          // Override file name to be just the base name
          Object.defineProperty(file, 'name', {
            value: baseName,
            writable: false,
            configurable: true
          });
          
          startUpload(singleFileList, fullPath, bucketId);
        }
      } catch (error) {
        console.error('Error creating folders:', error);
      }
    },
    [startUpload, path, bucketId]
  );

  const handleDrop = useCallback(
    async (e: DragEvent) => {
      e.preventDefault();
      e.stopPropagation();
      
      setIsDragOver(false);
      setDragCounter(0);

      const items = e.dataTransfer?.items;
      if (items && items.length > 0) {
        const allFiles: File[] = [];
        
        // Process all dropped items (files and directories)
        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          if (item.kind === 'file') {
            const entry = item.webkitGetAsEntry();
            if (entry) {
              const files = await processDirectoryEntry(entry);
              allFiles.push(...files);
            }
          }
        }
        
        if (allFiles.length > 0) {
          await createFoldersAndUploadFiles(allFiles);
        }
      }
    },
    [processDirectoryEntry, createFoldersAndUploadFiles]
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
                <FileIcon className="h-8 w-8 absolute -top-1 -right-1" />
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