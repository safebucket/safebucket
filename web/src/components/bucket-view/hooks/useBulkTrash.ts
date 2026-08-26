import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import type { RowSelectionState } from "@tanstack/react-table";
import type { BucketItem, IBucket } from "@/types/bucket.ts";
import { errorToast } from "@/lib/toast";
import { removeBucketItemsFromCache } from "@/queries/bucket";
import { api } from "@/lib/api";
import {toast} from "sonner";

interface IUseBulkTrashArgs {
  bucketId: string;
  items: Array<BucketItem>;
  rowSelection: RowSelectionState;
  clearRowSelection: () => void;
}

const isFolder = (item: BucketItem): boolean => !("size" in item);

export const useBulkTrash = ({
  bucketId,
  items,
  rowSelection,
  clearRowSelection,
}: IUseBulkTrashArgs) => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const selectedItems = useMemo(
    () => items.filter((item) => rowSelection[item.id]),
    [items, rowSelection],
  );

  const mutation = useMutation({
    mutationFn: (selected: Array<BucketItem>) => {
      const folderIds = selected.filter(isFolder).map((item) => item.id);
      const fileIds = selected
        .filter((item) => !isFolder(item))
        .map((item) => item.id);
      return api.post(`/buckets/${bucketId}/files/trash`, {
        folder_ids: folderIds,
        file_ids: fileIds,
      });
    },
    onMutate: async (selected: Array<BucketItem>) => {
      await queryClient.cancelQueries({ queryKey: ["buckets", bucketId] });
      const previous = removeBucketItemsFromCache(
        queryClient,
        bucketId,
        new Set(selected.map((item) => item.id)),
      );
      clearRowSelection();
      return { previous };
    },
    onSuccess: (_data, selected) => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "trash"],
      });
      toast.success(t("bucket.bulk_trash.success", { count: selected.length }));
    },
    onError: (err, _selected, context) => {
      if (context?.previous) {
        queryClient.setQueryData<IBucket>(
          ["buckets", bucketId],
          context.previous,
        );
      }
      errorToast(err as Error);
    },
  });

  return {
    run: () => mutation.mutate(selectedItems),
    selectedCount: selectedItems.length,
  };
};
