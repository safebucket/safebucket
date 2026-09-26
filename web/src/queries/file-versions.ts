import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import type { IFileVersion } from "@/types/file";
import { api } from "@/lib/api";
import i18n from "@/lib/i18n";
import { errorToast } from "@/lib/toast";

export type FileVersionAction = "restore" | "delete";

export const fileVersionsQueryOptions = (bucketId: string, fileId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "files", fileId, "versions"],
    queryFn: () =>
      api.get<Array<IFileVersion>>(
        `/buckets/${bucketId}/files/${fileId}/versions`,
      ),
  });

export const getFileVersionDownload = (
  bucketId: string,
  fileId: string,
  versionId: string,
) =>
  api.get<{ url: string }>(
    `/buckets/${bucketId}/files/${fileId}/versions/${versionId}/url`,
  );

export const useFileVersionMutation = (bucketId: string, fileId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      action,
      versionId,
    }: {
      action: FileVersionAction;
      versionId: string;
    }) =>
      action === "restore"
        ? api.put(`/buckets/${bucketId}/files/${fileId}`, {
            version_id: versionId,
          })
        : api.delete(
            `/buckets/${bucketId}/files/${fileId}/versions/${versionId}`,
          ),
    onSuccess: (_, { action }) => {
      toast.success(
        action === "restore"
          ? i18n.t("file_versions.restored")
          : i18n.t("file_versions.deleted"),
      );
    },
    onError: (error) => errorToast(error),
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] }),
        queryClient.invalidateQueries({ queryKey: ["buckets"], exact: true }),
      ]),
  });
};
