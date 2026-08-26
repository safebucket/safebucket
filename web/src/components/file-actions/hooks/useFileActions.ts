import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { toast } from "sonner";
import type { IFileActions } from "@/components/file-actions/helpers/types";
import type { IBucket } from "@/types/bucket.ts";
import {
  api_downloadFile,
  downloadFromStorage,
} from "@/components/file-actions/helpers/api";
import { errorToast } from "@/lib/toast";
import { removeBucketItemsFromCache } from "@/queries/bucket";
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
      toast.success(`Folder ${variables.name} has been created.`);
    },
  });

  const deleteFileMutation = useMutation({
    mutationFn: deleteFileMutationFn,
    onMutate: async ({ fileId }) => {
      await queryClient.cancelQueries({ queryKey: ["buckets", bucketId] });
      const previous = removeBucketItemsFromCache(
        queryClient,
        bucketId,
        new Set([fileId]),
      );
      return { previous };
    },
    onSuccess: ({ filename }) => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "trash"],
      });
      if (filename) {
        toast.success(`File "${filename}" has been moved to trash.`);
      }
    },
    onError: (err, _variables, context) => {
      if (context?.previous) {
        queryClient.setQueryData<IBucket>(
          ["buckets", bucketId],
          context.previous,
        );
      }
      errorToast(err as Error);
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
