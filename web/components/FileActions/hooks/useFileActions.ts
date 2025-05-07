import { mutate } from "swr";

import {
  api_deleteFile,
  api_downloadFile,
  downloadFromStorage,
} from "@/components/FileActions/helpers/api";
import { IFileActions } from "@/components/FileActions/helpers/types";
import { FileType } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { api_createFile } from "@/components/upload/helpers/api";

export const useFileActions = (): IFileActions => {
  const { bucketId, path } = useBucketViewContext();

  const createFolder = (name: string) => {
    api_createFile(name, FileType.folder, path, bucketId)
      .then(async (_) => {
        mutate(`/buckets/${bucketId}`).then(() =>
          successToast(`Folder ${name} has been created.`),
        );
      })
      .catch(errorToast);
  };

  const downloadFile = (fileId: string, filename: string) => {
    api_downloadFile(bucketId, fileId).then((res) =>
      downloadFromStorage(res.url, filename),
    );
  };

  const deleteFile = (fileId: string, filename: string) => {
    api_deleteFile(bucketId, fileId)
      .then(async () => {
        mutate(`/buckets/${bucketId}`).then(() =>
          successToast(`File ${filename} has been deleted.`),
        );
      })
      .catch(errorToast);
  };

  return {
    createFolder,
    deleteFile,
    downloadFile,
  };
};
