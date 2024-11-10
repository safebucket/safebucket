import { mutate } from "swr";

import {
  api_deleteFile,
  api_downloadFile,
  downloadFromStorage,
} from "@/components/FileActions/helpers/api";
import { IFileActions } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { toast } from "@/components/common/hooks/use-toast";

export const useFileActions = (): IFileActions => {
  const { bucketId } = useBucketViewContext();

  const downloadFile = (fileId: string, filename: string) => {
    api_downloadFile(fileId).then((res) =>
      downloadFromStorage(res.url, filename),
    );
  };

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
    downloadFile,
  };
};
