import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { api } from "@/lib/api";
import { bucketTrashedFilesQueryOptions } from "@/queries/bucket";

// Union type for trashed items (can be file or folder)
export type TrashedItem = (IFile | IFolder) & { itemType: "file" | "folder" };

export interface ITrashActions {
  trashedItems: Array<TrashedItem>;
  isLoading: boolean;
  restoreItem: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void;
  purgeItem: (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => void;
}

export const useTrashActions = (): ITrashActions => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { bucketId } = useBucketViewContext();

  // Fetch trashed items using centralized query options
  const { data, isLoading } = useQuery(
    bucketTrashedFilesQueryOptions(bucketId),
  );

  // Combine files and folders into a single array with type markers
  const trashedItems: Array<TrashedItem> = [
    ...(data?.files || []).map((file) => ({
      ...file,
      itemType: "file" as const,
    })),
    ...(data?.folders || []).map((folder) => ({
      ...folder,
      itemType: "folder" as const,
    })),
  ];

  // Restore item mutation (handles both files and folders)
  const restoreItemMutation = useMutation({
    mutationFn: ({
      itemId,
      itemType,
    }: {
      itemId: string;
      itemName: string;
      itemType: "file" | "folder";
    }) => {
      if (itemType === "file") {
        return api.patch(`/buckets/${bucketId}/files/${itemId}`, {
          status: "uploaded",
        });
      } else {
        return api.patch(`/buckets/${bucketId}/folders/${itemId}`, {
          status: "uploaded",
        });
      }
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "trash"],
      });
      queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
      successToast(
        t("bucket.trash_view.restore_success", {
          fileName: variables.itemName,
        }),
      );
    },
    onError: (error: Error) => errorToast(error),
  });

  const purgeItemMutation = useMutation({
    mutationFn: ({
      itemId,
      itemType,
    }: {
      itemId: string;
      itemName: string;
      itemType: "file" | "folder";
    }) => {
      if (itemType === "file") {
        return api.delete(`/buckets/${bucketId}/files/${itemId}`);
      } else {
        return api.delete(`/buckets/${bucketId}/folders/${itemId}`);
      }
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "trash"],
      });
      successToast(
        t("bucket.trash_view.purge_success", { fileName: variables.itemName }),
      );
    },
    onError: (error: Error) => errorToast(error),
  });

  const restoreItem = (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => {
    restoreItemMutation.mutate({ itemId, itemName, itemType });
  };

  const purgeItem = (
    itemId: string,
    itemName: string,
    itemType: "file" | "folder",
  ) => {
    purgeItemMutation.mutate({ itemId, itemName, itemType });
  };

  return {
    trashedItems,
    isLoading,
    restoreItem,
    purgeItem,
  };
};
