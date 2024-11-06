import { mutate } from "swr";

import { api_deleteFile } from "@/components/FileActions/helpers/api";
import { IFileActions } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { toast } from "@/components/common/hooks/use-toast";

export const useFileActions = (): IFileActions => {
  const { bucketId } = useBucketViewContext();

  const deleteFile = (fileId: string, filename: string) => {
    api_deleteFile(fileId).then(async () => {
      mutate(`/buckets/${bucketId}`).then(() =>
        toast({
          variant: "success",
          title: "Success",
          description: `File ${filename} has been deleted.`,
        }),
      );
    });
  };

  return {
    deleteFile,
  };
};
