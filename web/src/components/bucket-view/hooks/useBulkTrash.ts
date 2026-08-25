import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { RowSelectionState } from "@tanstack/react-table";
import type { BucketItem } from "@/types/bucket.ts";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { api } from "@/lib/api";

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
  const [isRunning, setIsRunning] = useState(false);

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
    onMutate: () => setIsRunning(true),
    onSettled: () => setIsRunning(false),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
      successToast(
        t("bucket.bulk_trash.success", { count: selectedItems.length }),
      );
      clearRowSelection();
    },
    onError: (err) => {
      errorToast(err as Error);
    },
  });

  return {
    run: () => mutation.mutate(selectedItems),
    isRunning,
    selectedCount: selectedItems.length,
  };
};
