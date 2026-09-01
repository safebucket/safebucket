import React, { useCallback, useEffect, useRef, useState } from "react";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  IUpload,
  IUploadPresign,
} from "@/components/upload/helpers/types";
import { generateRandomString } from "@/lib/utils";
import { api } from "@/lib/api";
import { uploadToStorage } from "@/components/upload/helpers/upload-engine";
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

interface UploadRuntime {
  controller: AbortController;
  target?: { bucketId: string; fileId: string };
  cleanupStarted: boolean;
}

export const UploadProvider = ({ children }: { children: React.ReactNode }) => {
  const queryClient = useQueryClient();
  const [uploads, setUploads] = useState<Array<IUpload>>([]);
  const runtimesRef = useRef<Map<string, UploadRuntime>>(new Map());
  const queueRef = useRef<Array<UploadTask>>([]);
  const activeRef = useRef(0);
  const invalidateTimersRef = useRef<
    Map<string, ReturnType<typeof setTimeout>>
  >(new Map());

  const cleanupUploadTarget = (runtime: UploadRuntime) => {
    if (!runtime.target || runtime.cleanupStarted) return;

    runtime.cleanupStarted = true;
    const { bucketId, fileId } = runtime.target;
    void api.delete(`/buckets/${bucketId}/files/${fileId}`).catch(() => {});
  };

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
    }: UploadTask) => {
      const abortController = new AbortController();
      const runtime: UploadRuntime = {
        controller: abortController,
        cleanupStarted: false,
      };
      runtimesRef.current.set(uploadId, runtime);

      try {
        const presignedUpload = await api.post<IUploadPresign>(
          `/buckets/${bucketId}/files`,
          {
            name: file.name,
            size: file.size,
            folder_id: folderId,
            expires_at: expiresAt,
          },
        );
        runtime.target = {
          bucketId,
          fileId: presignedUpload.id,
        };

        if (abortController.signal.aborted) cleanupUploadTarget(runtime);
        abortController.signal.throwIfAborted();

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

        abortController.signal.throwIfAborted();

        const config = await queryClient.ensureQueryData(configQueryOptions());
        const isMultipart =
          presignedUpload.method === "put" && presignedUpload.parts.length > 1;
        if (isMultipart || config.requiresUploadConfirmation) {
          abortController.signal.throwIfAborted();
          await api.patch(`/buckets/${bucketId}/files/${presignedUpload.id}`, {
            status: "uploaded",
          });
        }

        abortController.signal.throwIfAborted();

        return { uploadId, bucketId };
      } finally {
        runtimesRef.current.delete(uploadId);
      }
    },
    onSuccess: ({ uploadId, bucketId }) => {
      setUploads((prev) =>
        prev.map((u) =>
          u.id === uploadId && u.status === "uploading"
            ? { ...u, status: "success" }
            : u,
        ),
      );

      scheduleInvalidate(bucketId);
    },
    onError: (error: Error, { uploadId }) => {
      setUploads((prev) =>
        prev.map((u) =>
          u.id === uploadId && u.status === "uploading"
            ? { ...u, status: "error", error }
            : u,
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
      setUploads((prev) =>
        prev.map((u) =>
          u.id === task.uploadId ? { ...u, status: "uploading" } : u,
        ),
      );
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

        setUploads((prev) => [
          ...prev,
          {
            id: uploadId,
            name: file.name,
            path: file.name,
            progress: 0,
            status: "queued",
            size: file.size,
          },
        ]);

        const task: UploadTask = {
          file,
          bucketId,
          folderId,
          uploadId,
          expiresAt,
        };
        queueRef.current.push(task);
      });

      pump();
    },
    [pump],
  );

  const cancelUpload = useCallback((uploadId: string) => {
    queueRef.current = queueRef.current.filter((t) => t.uploadId !== uploadId);

    const runtime = runtimesRef.current.get(uploadId);
    if (runtime) {
      runtime.controller.abort();
      cleanupUploadTarget(runtime);
    }

    setUploads((prev) =>
      prev.map((u) =>
        u.id === uploadId
          ? { ...u, status: "cancelled", progress: 0, error: undefined }
          : u,
      ),
    );
  }, []);

  const clearUploads = useCallback(() => {
    setUploads((prev) =>
      prev.filter((u) => u.status === "uploading" || u.status === "queued"),
    );
  }, []);

  const hasActiveUploads = uploads.some(
    (upload) => upload.status === "uploading" || upload.status === "queued",
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
