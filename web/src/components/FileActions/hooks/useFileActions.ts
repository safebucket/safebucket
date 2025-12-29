import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { IFileActions } from "@/components/FileActions/helpers/types";
import {
  api_downloadFile,
  downloadFromStorage,
} from "@/components/FileActions/helpers/api";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { errorToast, successToast, toast } from "@/components/ui/hooks/use-toast";
import {
  createFolderMutationFn,
  deleteFileMutationFn,
} from "@/components/upload/helpers/api.ts";

export const useFileActions = (): IFileActions => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { bucketId, folderId } = useBucketViewContext();

  const createFolderMutation = useMutation({
    mutationFn: createFolderMutationFn,
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      successToast(`Folder ${variables.name} has been created.`);
    },
    onError: (error: Error) => errorToast(error),
  });

  const deleteFileMutation = useMutation({
    mutationFn: deleteFileMutationFn,
    onSuccess: ({ filename }) => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      if (filename) {
        successToast(`File "${filename}" has been moved to trash.`);
      }
    },
    onError: (error: Error) => errorToast(error),
  });

  const createFolder = (name: string) => {
    createFolderMutation.mutate({
      name,
      folderId,
      bucketId,
    });
  };

  const downloadFile = async (fileId: string, filename: string) => {
    try {
      const res = await api_downloadFile(bucketId, fileId);
      downloadFromStorage(res.url, filename);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";

      // Handle specific sharing-related errors
      if (errorMessage === "FILE_EXPIRED") {
        toast({
          variant: "destructive",
          title: t("download.error.title"),
          description: t("download.error.file_expired"),
        });
      } else if (errorMessage === "DOWNLOAD_LIMIT_REACHED") {
        toast({
          variant: "destructive",
          title: t("download.error.title"),
          description: t("download.error.limit_reached"),
        });
      } else {
        errorToast(error instanceof Error ? error : new Error(errorMessage));
      }
    }
  };

  const deleteFile = (fileId: string, filename: string, isFolder = false) => {
    deleteFileMutation.mutate({ bucketId, fileId, filename, isFolder });
  };

  return {
    createFolder,
    deleteFile,
    downloadFile,
  };
};
