import { useEffect, useRef, useState } from "react";

import { Check, ChevronDown, Upload, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Button } from "@/components/ui/button";
import { UploadRow } from "@/components/upload/components/UploadRow";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { cn, formatFileSize } from "@/lib/utils";

export const UploadPanel: FC = () => {
  const { t } = useTranslation();
  const { uploads, cancelUpload, clearUploads } = useUploadContext();
  const [isExpanded, setIsExpanded] = useState(true);
  const prevCountRef = useRef(0);

  const {
    successCount,
    failedCount,
    cancelledCount,
    activeCount,
    totalBytes,
    bytesUploaded,
  } = uploads.reduce(
    (summary, upload) => {
      const size = upload.size ?? 0;

      if (upload.status === "success") {
        summary.successCount += 1;
        summary.totalBytes += size;
        summary.bytesUploaded += size;
      } else if (upload.status === "error") {
        summary.failedCount += 1;
      } else if (upload.status === "cancelled") {
        summary.cancelledCount += 1;
      } else {
        summary.activeCount += 1;
        summary.totalBytes += size;
        if (upload.status === "uploading") {
          summary.bytesUploaded += (size * upload.progress) / 100;
        }
      }

      return summary;
    },
    {
      successCount: 0,
      failedCount: 0,
      cancelledCount: 0,
      activeCount: 0,
      totalBytes: 0,
      bytesUploaded: 0,
    },
  );
  const isComplete = uploads.length > 0 && activeCount === 0;

  useEffect(() => {
    if (uploads.length > prevCountRef.current) {
      setIsExpanded(true);
    }
    prevCountRef.current = uploads.length;
  }, [uploads.length]);

  if (uploads.length === 0) {
    return null;
  }

  const cancelAll = () => {
    uploads
      .filter((u) => u.status === "uploading" || u.status === "queued")
      .forEach((u) => cancelUpload(u.id));
  };

  return (
    <div className="fixed inset-x-4 bottom-8 z-50 mx-auto max-w-96 overflow-hidden rounded-xl border bg-card text-card-foreground shadow-lg md:inset-x-auto md:right-8 md:mx-0 md:w-96 md:max-w-lg">
      <div className="flex items-center gap-3 px-4 py-3">
        <div
          className={cn(
            "grid size-9 shrink-0 place-items-center rounded-lg",
            isComplete
              ? "bg-success-subtle text-success"
              : "bg-primary/10 text-primary",
          )}
        >
          {isComplete ? (
            <Check className="size-5" />
          ) : (
            <Upload className="size-5" />
          )}
        </div>

        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold">
            {isComplete
              ? t("upload.uploaded_summary", {
                  done: successCount,
                  total: uploads.length,
                })
              : t("upload.uploading_files", { count: activeCount })}
          </p>
          <p className="text-muted-foreground truncate text-xs">
            {isComplete
              ? failedCount > 0
                ? t("upload.files_could_not_upload", { count: failedCount })
                : cancelledCount > 0
                  ? t("upload.uploads_cancelled", { count: cancelledCount })
                  : t("upload.all_uploaded")
              : t("upload.bytes_of_total", {
                  uploaded: formatFileSize(bytesUploaded),
                  total: formatFileSize(totalBytes),
                })}
          </p>
        </div>

        <Button
          variant="ghost"
          size="icon-sm"
          className="text-muted-foreground shrink-0"
          onClick={() => setIsExpanded((prev) => !prev)}
          aria-label={t(isExpanded ? "upload.collapse" : "upload.expand")}
        >
          <ChevronDown
            className={cn("transition-transform", !isExpanded && "rotate-180")}
          />
        </Button>
        {isComplete && (
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground shrink-0"
            onClick={clearUploads}
            aria-label={t("upload.dismiss")}
          >
            <X />
          </Button>
        )}
      </div>

      {isExpanded && (
        <>
          <div className="max-h-72 divide-y overflow-y-auto border-t md:max-h-96">
            {uploads.map((upload) => (
              <UploadRow
                key={upload.id}
                upload={upload}
                isComplete={isComplete}
                onCancel={cancelUpload}
              />
            ))}
          </div>

          <div className="bg-muted/40 flex items-center justify-between gap-2 border-t px-4 py-3">
            {isComplete ? (
              <>
                <p className="text-muted-foreground text-xs">
                  {t("upload.uploaded_footer", { count: successCount })}
                  {failedCount > 0 && (
                    <>
                      {" · "}
                      <span className="text-destructive font-medium">
                        {t("upload.failed_footer", { count: failedCount })}
                      </span>
                    </>
                  )}
                  {cancelledCount > 0 && (
                    <>
                      {" · "}
                      {t("upload.cancelled_footer", { count: cancelledCount })}
                    </>
                  )}
                </p>
                <Button
                  size="sm"
                  className="rounded-full"
                  onClick={() => clearUploads()}
                >
                  {t("upload.done")}
                </Button>
              </>
            ) : (
              <>
                <p className="text-muted-foreground text-xs">
                  <span className="text-foreground font-medium">
                    {successCount}
                  </span>{" "}
                  {t("upload.of_total_done", { total: uploads.length })}
                </p>
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full"
                  onClick={cancelAll}
                >
                  <X />
                  {t("upload.cancel_all")}
                </Button>
              </>
            )}
          </div>
        </>
      )}
    </div>
  );
};
