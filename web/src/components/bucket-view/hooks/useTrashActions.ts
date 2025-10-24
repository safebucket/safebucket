import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import {
  api_purgeFile,
  api_restoreFile,
} from "@/components/bucket-view/helpers/api";
import { bucketTrashedFilesQueryOptions } from "@/queries/bucket";
import type { IFile } from "@/types/file.ts";

export interface ITrashActions {
  trashedFiles: IFile[];
  isLoading: boolean;
  restoreFile: (fileId: string, fileName: string) => void;
  purgeFile: (fileId: string, fileName: string) => void;
}

export const useTrashActions = (): ITrashActions => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { bucketId } = useBucketViewContext();

  // Fetch trashed files using centralized query options
  const {
    data: trashedFiles = [],
    isLoading,
  } = useQuery(bucketTrashedFilesQueryOptions(bucketId));

  // Restore file mutation
  const restoreFileMutation = useMutation({
    mutationFn: ({ fileId }: { fileId: string; fileName: string }) =>
      api_restoreFile(bucketId, fileId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId, "trash"] });
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
      successToast(t("bucket.trash_view.restore_success", { fileName: variables.fileName }));
    },
    onError: (error: Error) => errorToast(error),
  });

  const purgeFileMutation = useMutation({
    mutationFn: ({ fileId }: { fileId: string; fileName: string }) =>
      api_purgeFile(bucketId, fileId),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId, "trash"] });
      successToast(t("bucket.trash_view.purge_success", { fileName: variables.fileName }));
    },
    onError: (error: Error) => errorToast(error),
  });

  const restoreFile = (fileId: string, fileName: string) => {
    restoreFileMutation.mutate({ fileId, fileName });
  };

  const purgeFile = (fileId: string, fileName: string) => {
    purgeFileMutation.mutate({ fileId, fileName });
  };

  return {
    trashedFiles,
    isLoading,
    restoreFile,
    purgeFile,
  };
};
