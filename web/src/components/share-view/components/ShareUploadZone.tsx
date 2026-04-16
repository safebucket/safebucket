import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChevronDown, Upload, X } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import type React from "react";
import type { DragEvent, FC } from "react";

import type { IUpload } from "@/components/upload/helpers/types";
import type { IPublicShareResponse } from "@/types/share";
import { FileStatus } from "@/types/file";
import { extractFilesFromDrop } from "@/components/upload/helpers/file-processing";
import { uploadToStorage } from "@/components/upload/helpers/api";
import {
  getStatusIcon,
  getStatusText,
} from "@/components/upload/helpers/utils";
import {
  useShareConfirmUploadMutation,
  useShareUploadMutation,
} from "@/queries/share";
import { configQueryOptions } from "@/queries/config";
import { cn, formatFileSize, generateRandomString } from "@/lib/utils";
import { errorToast } from "@/components/ui/hooks/use-toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";

export type ShareUploadHandler = (files: Array<File>) => void;

interface IShareUploadZoneProps {
  shareId: string;
  token: string | null;
  maxUploadSize: number | null;
  folderId?: string;
  onReady: (uploadFiles: ShareUploadHandler) => void;
  children: React.ReactNode;
}

export const ShareUploadZone: FC<IShareUploadZoneProps> = ({
  shareId,
  token,
  maxUploadSize,
  folderId,
  onReady,
  children,
}) => {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [isDragOver, setIsDragOver] = useState(false);
  const [uploads, setUploads] = useState<Array<IUpload>>([]);
  const [isPanelExpanded, setIsPanelExpanded] = useState(true);
  const dragCounterRef = useRef(0);
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());

  const uploadMutation = useShareUploadMutation(shareId, token);
  const confirmMutation = useShareConfirmUploadMutation(shareId, token);

  const processUpload = useCallback(
    async (file: File) => {
      if (maxUploadSize && file.size > maxUploadSize) {
        errorToast(
          new Error(
            t("share_consumer.upload_size_exceeded", {
              max: formatFileSize(maxUploadSize),
            }),
          ),
        );
        return;
      }

      const uploadId = generateRandomString(12);
      const abortController = new AbortController();
      abortControllersRef.current.set(uploadId, abortController);

      setUploads((prev) => [
        ...prev,
        {
          id: uploadId,
          name: file.name,
          path: file.name,
          progress: 0,
          status: "uploading",
        },
      ]);

      try {
        const presigned = await uploadMutation.mutateAsync({
          name: file.name,
          size: file.size,
          folder_id: folderId,
        });

        await uploadToStorage(
          {
            id: presigned.id,
            url: presigned.url,
            body: presigned.body ?? {},
            path: "",
          },
          file,
          (progress) => {
            setUploads((prev) =>
              prev.map((u) => (u.id === uploadId ? { ...u, progress } : u)),
            );
          },
          abortController.signal,
        );

        const config = await queryClient.ensureQueryData(configQueryOptions());
        if (config.requiresUploadConfirmation) {
          await confirmMutation.mutateAsync(presigned.id);
        }

        setUploads((prev) =>
          prev.map((u) =>
            u.id === uploadId ? { ...u, status: "success" } : u,
          ),
        );

        const extension = file.name.includes(".")
          ? (file.name.split(".").pop() ?? "")
          : "";

        queryClient.setQueriesData<IPublicShareResponse>(
          { queryKey: ["shares", shareId, "content"] },
          (old) => {
            if (!old) return old;
            return {
              ...old,
              current_uploads: old.current_uploads + 1,
              files: [
                ...old.files,
                {
                  id: presigned.id,
                  name: file.name,
                  size: file.size,
                  extension,
                  folder_id: folderId,
                  status: FileStatus.uploaded,
                  created_at: new Date().toISOString(),
                  deleted_at: null,
                  expires_at: null,
                },
              ],
            };
          },
        );
      } catch (err) {
        setUploads((prev) =>
          prev.map((u) =>
            u.id === uploadId
              ? { ...u, status: "error", error: err as Error }
              : u,
          ),
        );
        if (err instanceof Error && err.message !== "Upload cancelled") {
          errorToast(err);
        }
      } finally {
        abortControllersRef.current.delete(uploadId);
      }
    },
    [
      shareId,
      token,
      folderId,
      maxUploadSize,
      uploadMutation,
      confirmMutation,
      queryClient,
      t,
    ],
  );

  const handleFiles = useCallback(
    (files: Array<File>) => {
      files.forEach((file) => processUpload(file));
    },
    [processUpload],
  );

  onReady(handleFiles);

  const handleDragEnter = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current += 1;
    if (e.dataTransfer.types.includes("Files")) {
      setIsDragOver(true);
    }
  }, []);

  const handleDragLeave = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current -= 1;
    if (dragCounterRef.current <= 0) {
      dragCounterRef.current = 0;
      setIsDragOver(false);
    }
  }, []);

  const handleDragOver = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.dataTransfer.types.includes("Files")) {
      e.dataTransfer.dropEffect = "copy";
    }
  }, []);

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragOver(false);
      dragCounterRef.current = 0;

      const filesWithPaths = await extractFilesFromDrop(e.dataTransfer);
      if (filesWithPaths.length > 0) {
        handleFiles(filesWithPaths.map((f) => f.file));
      }
    },
    [handleFiles],
  );

  const cancelUpload = useCallback((uploadId: string) => {
    const controller = abortControllersRef.current.get(uploadId);
    if (controller) {
      controller.abort();
      abortControllersRef.current.delete(uploadId);
    }
    setUploads((prev) => prev.filter((u) => u.id !== uploadId));
  }, []);

  const clearUploads = useCallback(() => {
    setUploads((prev) => prev.filter((u) => u.status === "uploading"));
  }, []);

  const completedCount = uploads.filter((u) => u.status === "success").length;
  const failedCount = uploads.filter((u) => u.status === "error").length;
  const hasFinished = completedCount > 0 || failedCount > 0;

  return (
    <div
      className={cn("relative")}
      onDragEnter={handleDragEnter}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}

      {uploads.length > 0 && (
        <div className="fixed inset-x-4 bottom-8 z-50 mx-auto max-w-96 rounded-lg border bg-card text-card-foreground shadow-lg md:inset-x-auto md:right-8 md:mx-0">
          <button
            type="button"
            className="flex w-full items-center justify-between px-4 py-3"
            onClick={() => setIsPanelExpanded((prev) => !prev)}
          >
            <div className="flex items-center gap-2">
              <Upload className="h-4 w-4" />
              <span className="text-sm font-semibold">
                {t("upload.uploads")}
              </span>
              <Badge className="h-5 min-w-5 justify-center text-xs">
                {uploads.length}
              </Badge>
            </div>
            <ChevronDown
              className={`h-4 w-4 transition-transform ${isPanelExpanded ? "" : "rotate-180"}`}
            />
          </button>

          {isPanelExpanded && (
            <div className="border-t px-4 pb-4">
              {hasFinished && (
                <div className="mt-3 flex items-center justify-between">
                  <div className="text-xs">
                    {completedCount > 0 && (
                      <span className="text-green-600">
                        {completedCount} {t("upload.completed")}
                      </span>
                    )}
                    {completedCount > 0 && failedCount > 0 && " · "}
                    {failedCount > 0 && (
                      <span className="text-red-600">
                        {failedCount} {t("upload.failed")}
                      </span>
                    )}
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="text-muted-foreground h-auto px-2 py-1 text-xs"
                    onClick={(e) => {
                      e.stopPropagation();
                      clearUploads();
                    }}
                  >
                    {t("upload.clear_all")}
                  </Button>
                </div>
              )}

              <div className="mt-2 max-h-64 space-y-1 overflow-y-auto">
                {uploads.map((upload) => (
                  <div
                    key={upload.id}
                    className="hover:bg-muted/30 flex items-center gap-3 rounded p-2"
                  >
                    <div className="shrink-0">
                      {getStatusIcon(upload.status, upload.progress)}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div
                        className="truncate text-sm font-medium"
                        title={upload.path}
                      >
                        {upload.name}
                      </div>
                      <div className="flex items-center gap-2">
                        {upload.status === "uploading" && (
                          <Progress
                            value={upload.progress}
                            className="h-2 flex-1"
                          />
                        )}
                        <div className="text-muted-foreground text-xs whitespace-nowrap">
                          {getStatusText(upload.status, upload.progress, t)}
                        </div>
                      </div>
                    </div>
                    {upload.status === "uploading" && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={() => cancelUpload(upload.id)}
                        aria-label={`Cancel upload: ${upload.name}`}
                      >
                        <X className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {isDragOver && (
        <div className="bg-primary/10 border-primary fixed inset-0 z-50 flex items-center justify-center border-4 border-dashed">
          <div className="text-primary flex flex-col items-center justify-center space-y-4">
            <div className="relative">
              <Upload className="h-20 w-20" />
            </div>
            <div className="text-center">
              <p className="text-xl font-semibold">
                {t("share_consumer.drop_files")}
              </p>
              <p className="text-primary/80 text-base">
                {t("share_consumer.drop_description")}
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
