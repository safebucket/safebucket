import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { api } from "@/lib/api";
import { errorToast } from "@/lib/toast";

interface IMoveResponse {
  moved_files: number;
  moved_folders: number;
  unchanged_items: number;
}

interface IMoveVariables {
  items: Array<BucketItem>;
  targetFolderId: string | undefined;
}

interface IUseMoveItemsOptions {
  onSuccess?: () => void;
}

const moveItems = async (
  bucketId: string,
  { items, targetFolderId }: IMoveVariables,
) => {
  const folders = items.filter(isFolder);
  const files = items.filter((item) => !isFolder(item));

  const response = await api.post<IMoveResponse>(`/buckets/${bucketId}/move`, {
    file_ids: files.map((file) => file.id),
    folder_ids: folders.map((folder) => folder.id),
    destination_folder_id: targetFolderId ?? null,
  });

  return {
    count:
      response.moved_files + response.moved_folders + response.unchanged_items,
  };
};

export function useMoveItems(
  bucketId: string,
  { onSuccess }: IUseMoveItemsOptions = {},
) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();

  return useMutation({
    mutationFn: (variables: IMoveVariables) => moveItems(bucketId, variables),
    onSuccess: async ({ count }) => {
      onSuccess?.();
      await queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
      toast.success(t("bucket.view.move_success", { count }));
    },
    onError: (error) => errorToast(error),
  });
}
