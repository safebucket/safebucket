import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import type {
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types";
import type { IFolder } from "@/types/folder";
import { createFolderMutationFn } from "@/components/upload/helpers/api";
import {
  extractFolderPaths,
  toStagedFiles,
} from "@/components/upload/helpers/file-processing";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

interface UseUploadDialogParams {
  bucketId: string;
  folderId: string | undefined;
}

export interface UseUploadDialogReturn {
  dialogProps: { open: boolean; onOpenChange: (open: boolean) => void };
  open: () => void;
  close: () => void;
  openWithFiles: (files: Array<FileWithPath>) => void;
  stagedFiles: Array<StagedFile>;
  addFiles: (files: Array<FileWithPath>) => void;
  removeFile: (id: string) => void;
  expiresAt: Date | undefined;
  setExpiresAt: (date: Date | undefined) => void;
  isAdvancedOpen: boolean;
  setIsAdvancedOpen: (open: boolean) => void;
  handleUpload: () => Promise<void>;
}

export const useUploadDialog = ({
  bucketId,
  folderId,
}: UseUploadDialogParams): UseUploadDialogReturn => {
  const queryClient = useQueryClient();
  const { startUpload } = useUploadContext();
  const [isOpen, setIsOpen] = useState(false);
  const [stagedFiles, setStagedFiles] = useState<Array<StagedFile>>([]);
  const [expiresAt, setExpiresAt] = useState<Date | undefined>(undefined);
  const [isAdvancedOpen, setIsAdvancedOpen] = useState(false);

  const resetState = useCallback(() => {
    setStagedFiles([]);
    setExpiresAt(undefined);
    setIsAdvancedOpen(false);
  }, []);

  const open = useCallback(() => {
    resetState();
    setIsOpen(true);
  }, [resetState]);

  const close = useCallback(() => {
    setIsOpen(false);
    resetState();
  }, [resetState]);

  const openWithFiles = useCallback(
    (files: Array<FileWithPath>) => {
      resetState();
      setStagedFiles(toStagedFiles(files));
      setIsOpen(true);
    },
    [resetState],
  );

  const onOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen) {
        open();
      } else {
        close();
      }
    },
    [open, close],
  );

  const addFiles = useCallback((files: Array<FileWithPath>) => {
    setStagedFiles((prev) => [...prev, ...toStagedFiles(files)]);
  }, []);

  const removeFile = useCallback((id: string) => {
    setStagedFiles((prev) => prev.filter((f) => f.id !== id));
  }, []);

  const handleUpload = useCallback(async () => {
    const expiresAtIso = expiresAt ? expiresAt.toISOString() : null;

    const hasNestedFiles = stagedFiles.some((f) => f.relativePath !== "");
    const pathToIdMap = new Map<string, string | undefined>();
    pathToIdMap.set("", folderId);

    if (hasNestedFiles) {
      const folderPaths = extractFolderPaths(
        stagedFiles.map((f) => ({
          file: f.file,
          relativePath: f.relativePath,
        })),
      );

      for (const folderPath of folderPaths) {
        const pathParts = folderPath.split("/");
        const folderName = pathParts.at(-1) ?? "";
        const parentPath = pathParts.slice(0, -1).join("/");
        const parentId = pathToIdMap.get(parentPath) ?? undefined;

        const folder: IFolder = await createFolderMutationFn({
          name: folderName,
          folderId: parentId,
          bucketId,
        });
        pathToIdMap.set(folderPath, folder.id);
      }

      await queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId],
      });
    }

    for (const staged of stagedFiles) {
      const targetFolderId = pathToIdMap.get(staged.relativePath) ?? folderId;
      startUpload([staged.file], bucketId, targetFolderId, expiresAtIso);
    }

    close();
  }, [
    stagedFiles,
    expiresAt,
    folderId,
    bucketId,
    startUpload,
    queryClient,
    close,
  ]);

  return {
    dialogProps: { open: isOpen, onOpenChange },
    open,
    close,
    openWithFiles,
    stagedFiles,
    addFiles,
    removeFile,
    expiresAt,
    setExpiresAt,
    isAdvancedOpen,
    setIsAdvancedOpen,
    handleUpload,
  };
};
