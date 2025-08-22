import React, { useReducer } from "react";

import { useQueryClient } from "@tanstack/react-query";
import * as actions from "../store/actions";
import { generateRandomString } from "@/lib/utils";

import { FileType } from "@/components/bucket-view/helpers/types";
import { successToast } from "@/components/ui/hooks/use-toast";
import {
  api_createFile,
  uploadToStorage,
} from "@/components/upload/helpers/api";
import { UploadStatus } from "@/components/upload/helpers/types";
import { UploadContext } from "@/components/upload/hooks/useUploadContext";
import { uploadsReducer } from "@/components/upload/store/reducer";

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const queryClient = useQueryClient();
  const [uploads, dispatch] = useReducer(uploadsReducer, []);

  const addUpload = (uploadId: string, filename: string, path: string) =>
    dispatch(actions.addUpload(uploadId, filename, path));

  const updateProgress = (uploadId: string, progress: number) =>
    dispatch(actions.updateProgress(uploadId, progress));

  const updateStatus = (uploadId: string, status: UploadStatus) =>
    dispatch(actions.updateStatus(uploadId, status));

  const startUpload = (files: FileList, path: string, bucketId: string) => {
    const file = files[0];
    const uploadId = generateRandomString(12);

    // Create full path display: path + filename
    const fullPath =
      path && path !== "/" ? `${path}/${file.name}` : `/${file.name}`;

    addUpload(uploadId, file.name, fullPath);

    // Ensure path is never empty - backend requires non-empty path
    const apiPath = path || "/";
    api_createFile(file.name, FileType.file, apiPath, bucketId, file.size).then(
      (presignedUpload) => {
        uploadToStorage(presignedUpload, file, uploadId, updateProgress).then(
          (success: boolean) => {
            const status = success ? UploadStatus.success : UploadStatus.failed;
            updateStatus(uploadId, status);

            if (success) {
              successToast(`Upload completed for ${file.name}`);
            }
            queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
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
