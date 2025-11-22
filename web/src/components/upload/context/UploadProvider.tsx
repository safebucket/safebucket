import React, { useEffect, useReducer } from "react";

import { useQueryClient } from "@tanstack/react-query";
import * as actions from "../store/actions";
import { generateRandomString } from "@/lib/utils";

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

  // Add beforeunload warning when uploads are in progress
  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      const hasActiveUploads = uploads.some(
        (upload) => upload.status === UploadStatus.uploading,
      );

      if (hasActiveUploads) {
        event.preventDefault();
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [uploads]);

  const addUpload = (uploadId: string, filename: string, displayPath: string) =>
    dispatch(actions.addUpload(uploadId, filename, displayPath));

  const updateProgress = (uploadId: string, progress: number) =>
    dispatch(actions.updateProgress(uploadId, progress));

  const updateStatus = (uploadId: string, status: UploadStatus) =>
    dispatch(actions.updateStatus(uploadId, status));

  const startUpload = (
    files: FileList,
    bucketId: string,
    folderId: string | null,
  ) => {
    const file = files[0];
    const uploadId = generateRandomString(12);

    // Display path for UI (just show filename for now, folder path can be added later if needed)
    const displayPath = file.name;

    addUpload(uploadId, file.name, displayPath);

    api_createFile(file.name, bucketId, file.size, folderId).then(
      (presignedUpload) => {
        queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
        uploadToStorage(presignedUpload, file, uploadId, updateProgress).then(
          (success: boolean) => {
            const status = success ? UploadStatus.success : UploadStatus.failed;
            updateStatus(uploadId, status);

            if (success) {
              setTimeout(function () {
                queryClient
                  .invalidateQueries({ queryKey: ["buckets", bucketId] })
                  .then(() =>
                    successToast(`Upload completed for ${file.name}`),
                  );
              }, 2000);
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
