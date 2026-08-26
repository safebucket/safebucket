import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { api } from "@/lib/api";

interface IMoveResult {
  id: string;
  status: string;
  code?: string;
}

interface IMoveResponse {
  results: Array<IMoveResult>;
}

interface IMoveVariables {
  items: Array<BucketItem>;
  targetFolderId: string | undefined;
}

const countFailed = (response: IMoveResponse) =>
  response.results.filter((result) => result.status !== "ok").length;

const moveItems = async (
  bucketId: string,
  { items, targetFolderId }: IMoveVariables,
) => {
  const folders = items.filter(isFolder);
  const files = items.filter((item) => !isFolder(item));

  const requests: Array<Promise<number>> = [];

  if (folders.length > 0) {
    requests.push(
      api
        .post<IMoveResponse>(`/buckets/${bucketId}/folders/move`, {
          ids: folders.map((folder) => folder.id),
          folder_id: targetFolderId ?? null,
        })
        .then(countFailed),
    );
  }

  if (files.length > 0) {
    requests.push(
      api
        .post<IMoveResponse>(`/buckets/${bucketId}/files/move`, {
          ids: files.map((file) => file.id),
          folder_id: targetFolderId ?? null,
        })
        .then(countFailed),
    );
  }

  const failures = await Promise.all(requests);
  const failed = failures.reduce((sum, count) => sum + count, 0);

  return { total: items.length, failed };
};

export function useMoveItems(bucketId: string) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();

  const mutation = useMutation({
    mutationFn: (variables: IMoveVariables) => moveItems(bucketId, variables),
    onSuccess: ({ total, failed }) => {
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
      if (failed > 0) {
        toast.warning(
          t("bucket.view.move_partial", { moved: total - failed, failed }),
        );
        return;
      }
      toast.success(t("bucket.view.move_success", { count: total }));
    },
    onError: () => {
      toast.error(t("errors.INTERNAL_SERVER_ERROR"));
    },
  });

  return (items: Array<BucketItem>, targetFolderId: string | undefined) =>
    mutation.mutate({ items, targetFolderId });
}
