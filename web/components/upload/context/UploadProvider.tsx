import React, { useReducer } from "react";

import { mutate } from "swr";

import { FileType } from "@/components/bucket-view/helpers/types";
import { successToast } from "@/components/ui/hooks/use-toast";
import {
  api_createFile,
  uploadToStorage,
} from "@/components/upload/helpers/api";
import { UploadStatus } from "@/components/upload/helpers/types";
import { UploadContext } from "@/components/upload/hooks/useUploadContext";
import { uploadsReducer } from "@/components/upload/store/reducer";

import * as actions from "../store/actions";

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const [uploads, dispatch] = useReducer(uploadsReducer, []);

  const addUpload = (uploadId: string, filename: string) =>
    dispatch(actions.addUpload(uploadId, filename));

  const updateProgress = (uploadId: string, progress: number) =>
    dispatch(actions.updateProgress(uploadId, progress));

  const updateStatus = (uploadId: string, status: UploadStatus) =>
    dispatch(actions.updateStatus(uploadId, status));

  const startUpload = async (
    files: FileList,
    path: string,
    bucketId: string,
  ) => {
    const file = files[0];
    const uploadId = crypto.randomUUID();

    addUpload(uploadId, file.name);

    api_createFile(file.name, FileType.file, path, bucketId, file.size).then(
      async (presignedUpload) => {
        await mutate(`/buckets/${bucketId}`);
        uploadToStorage(presignedUpload, file, uploadId, updateProgress).then(
          async (success: boolean) => {
            const status = success ? UploadStatus.success : UploadStatus.failed;
            updateStatus(uploadId, status);

            if (success) {
              successToast(`Upload completed for ${file.name}`)
            }
          },
        );
      },
    );
  };

  return (
    <UploadContext.Provider value={{ uploads, startUpload }}>
      {children}
    </UploadContext.Provider>
  );
};
