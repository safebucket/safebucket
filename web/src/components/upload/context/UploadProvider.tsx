import React, { useCallback, useEffect, useRef, useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IUpload } from "@/components/upload/helpers/types";
import { generateRandomString } from "@/lib/utils";

import { successToast } from "@/components/ui/hooks/use-toast";
import {
  api_confirmUpload,
  api_createFile,
  uploadToStorage,
} from "@/components/upload/helpers/api";
import { configQueryOptions } from "@/queries/config";
import { UploadContext } from "@/components/upload/hooks/useUploadContext";

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<Array<IUpload>>([]);
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());

  const uploadMutation = useMutation({
    mutationFn: async ({
      file,
      bucketId,
      folderId,
      uploadId,
    }: {
      file: File;
      bucketId: string;
      folderId: string | undefined;
      uploadId: string;
    }) => {
      const abortController = new AbortController();
      abortControllersRef.current.set(uploadId, abortController);

      try {
        const presignedUpload = await api_createFile(
          file.name,
          bucketId,
          file.size,
          folderId,
        );

        queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });

        await uploadToStorage(
          presignedUpload,
          file,
          (progress) => {
            setUploads((prev) =>
              prev.map((u) => (u.id === uploadId ? { ...u, progress } : u)),
            );
          },
          abortController.signal,
        );

        // Confirm upload if required by storage provider (e.g., generic S3 without bucket notifications)
        const config = await queryClient.ensureQueryData(configQueryOptions());
        if (config.requiresUploadConfirmation) {
          await api_confirmUpload(bucketId, presignedUpload.id);
        }

        return { uploadId, fileName: file.name, bucketId };
      } finally {
        abortControllersRef.current.delete(uploadId);
      }
    },
    onSuccess: ({ uploadId, fileName, bucketId }) => {
      setUploads((prev) =>
        prev.map((u) => (u.id === uploadId ? { ...u, status: "success" } : u)),
      );

      setTimeout(() => {
        queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
        successToast(`Upload completed for ${fileName}`);
      }, 1000);

      setTimeout(() => {
        setUploads((prev) => prev.filter((u) => u.id !== uploadId));
      }, 3000);
    },
    onError: (error: Error, { uploadId }) => {
      setUploads((prev) =>
        prev.map((u) =>
          u.id === uploadId ? { ...u, status: "error", error } : u,
        ),
      );
    },
  });

  const startUpload = useCallback(
    (files: FileList, bucketId: string, folderId: string | undefined) => {
      Array.from(files).forEach((file) => {
        const uploadId = generateRandomString(12);
        const displayPath = file.name;

        setUploads((prev) => [
          ...prev,
          {
            id: uploadId,
            name: file.name,
            path: displayPath,
            progress: 0,
            status: "uploading",
          },
        ]);

        uploadMutation.mutate({ file, bucketId, folderId, uploadId });
      });
    },
    [uploadMutation],
  );

  const cancelUpload = useCallback((uploadId: string) => {
    const abortController = abortControllersRef.current.get(uploadId);
    if (abortController) {
      abortController.abort();
      abortControllersRef.current.delete(uploadId);
    }

    setUploads((prev) => prev.filter((u) => u.id !== uploadId));
  }, []);

  const hasActiveUploads = uploads.some(
    (upload) => upload.status === "uploading",
  );

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (hasActiveUploads) {
        event.preventDefault();
        return "";
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [hasActiveUploads]);

  return (
    <UploadContext.Provider
      value={{ uploads, startUpload, cancelUpload, hasActiveUploads }}
    >
      {children}
    </UploadContext.Provider>
  );
};
