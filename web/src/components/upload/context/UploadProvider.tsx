import React, { useCallback, useEffect, useRef, useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { IUpload } from "@/components/upload/helpers/types";
import { generateRandomString } from "@/lib/utils";
import {
  api_confirmUpload,
  api_createFile,
  uploadToStorage,
} from "@/components/upload/helpers/api";
import { configQueryOptions } from "@/queries/config";
import { UploadContext } from "@/components/upload/hooks/useUploadContext";

const MAX_CONCURRENT_UPLOADS = 4;

interface UploadTask {
  file: File;
  bucketId: string;
  folderId: string | undefined;
  uploadId: string;
  expiresAt: string | null;
}

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<Array<IUpload>>([]);
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());
  const queueRef = useRef<Array<UploadTask>>([]);
  const activeRef = useRef(0);
  const invalidateTimersRef = useRef<
    Map<string, ReturnType<typeof setTimeout>>
  >(new Map());

  const scheduleInvalidate = useCallback(
    (bucketId: string) => {
      const timers = invalidateTimersRef.current;
      clearTimeout(timers.get(bucketId));
      timers.set(
        bucketId,
        setTimeout(() => {
          queryClient.invalidateQueries({ queryKey: ["buckets", bucketId] });
          timers.delete(bucketId);
        }, 1000),
      );
    },
    [queryClient],
  );

  const uploadMutation = useMutation({
    meta: { skipGlobalErrorToast: true },
    mutationFn: async ({
      file,
      bucketId,
      folderId,
      uploadId,
      expiresAt,
    }: {
      file: File;
      bucketId: string;
      folderId: string | undefined;
      uploadId: string;
      expiresAt: string | null;
    }) => {
      const abortController = new AbortController();
      abortControllersRef.current.set(uploadId, abortController);

      try {
        const presignedUpload = await api_createFile(
          file.name,
          bucketId,
          file.size,
          folderId,
          expiresAt,
        );

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
    onSuccess: ({ uploadId, bucketId }) => {
      setUploads((prev) =>
        prev.map((u) => (u.id === uploadId ? { ...u, status: "success" } : u)),
      );

      scheduleInvalidate(bucketId);
    },
    onError: (error: Error, { uploadId }) => {
      setUploads((prev) =>
        prev.map((u) =>
          u.id === uploadId ? { ...u, status: "error", error } : u,
        ),
      );
    },
  });

  const pump = useCallback(() => {
    while (
      activeRef.current < MAX_CONCURRENT_UPLOADS &&
      queueRef.current.length > 0
    ) {
      const task = queueRef.current.shift()!;
      activeRef.current += 1;
      uploadMutation
        .mutateAsync(task)
        .catch(() => {})
        .finally(() => {
          activeRef.current -= 1;
          pump();
        });
    }
  }, [uploadMutation]);

  const startUpload = useCallback(
    (
      files: Array<File>,
      bucketId: string,
      folderId: string | undefined,
      expiresAt: string | null,
    ) => {
      files.forEach((file) => {
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

        queueRef.current.push({
          file,
          bucketId,
          folderId,
          uploadId,
          expiresAt,
        });
      });

      pump();
    },
    [pump],
  );

  const cancelUpload = useCallback((uploadId: string) => {
    queueRef.current = queueRef.current.filter((t) => t.uploadId !== uploadId);

    const abortController = abortControllersRef.current.get(uploadId);
    if (abortController) {
      abortController.abort();
      abortControllersRef.current.delete(uploadId);
    }

    setUploads((prev) => prev.filter((u) => u.id !== uploadId));
  }, []);

  const clearUploads = useCallback(() => {
    setUploads((prev) => prev.filter((u) => u.status === "uploading"));
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
      value={{
        uploads,
        startUpload,
        cancelUpload,
        clearUploads,
        hasActiveUploads,
      }}
    >
      {children}
    </UploadContext.Provider>
  );
};
