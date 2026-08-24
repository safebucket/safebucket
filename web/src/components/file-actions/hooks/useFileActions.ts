import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import type { IFileActions } from "@/components/file-actions/helpers/types";
import {
  api_downloadFile,
  downloadFromStorage,
} from "@/components/file-actions/helpers/api";
import { successToast } from "@/components/ui/hooks/use-toast";
import {
  createFolderMutationFn,
  deleteFileMutationFn,
} from "@/components/upload/helpers/api.ts";

export const useFileActions = (): IFileActions => {
  const queryClient = useQueryClient();
  const { bucketId, folderId } = useParams({
    from: "/_authenticated/buckets/$bucketId/files/{-$folderId}",
  });

  const createFolderMutation = useMutation({
    mutationFn: createFolderMutationFn,
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      successToast(`Folder ${variables.name} has been created.`);
    },
  });

  const deleteFileMutation = useMutation({
    mutationFn: deleteFileMutationFn,
    onSuccess: ({ filename }) => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      if (filename) {
        successToast(`File "${filename}" has been moved to trash.`);
      }
    },
  });

  const createFolder = (name: string) => {
    createFolderMutation.mutate({
      name,
      folderId,
      bucketId,
    });
  };

  const downloadFile = (fileId: string, filename: string) => {
    api_downloadFile(bucketId, fileId).then((res) =>
      downloadFromStorage(res.url, filename),
    );
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
