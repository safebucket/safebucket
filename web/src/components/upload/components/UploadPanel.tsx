import { useState } from "react";

import { ChevronDown, Upload, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  getStatusIcon,
  getStatusText,
} from "@/components/upload/helpers/utils";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

export const UploadPanel: FC = () => {
  const { t } = useTranslation();
  const { uploads, cancelUpload, clearUploads } = useUploadContext();
  const [isExpanded, setIsExpanded] = useState(true);

  if (uploads.length === 0) {
    return null;
  }

  const completedCount = uploads.filter((u) => u.status === "success").length;
  const failedCount = uploads.filter((u) => u.status === "error").length;
  const hasFinished = completedCount > 0 || failedCount > 0;

  return (
    <div className="fixed right-4 bottom-8 z-50 w-96 rounded-lg border bg-card text-card-foreground shadow-lg">
      <button
        type="button"
        className="flex w-full items-center justify-between px-4 py-3"
        onClick={() => setIsExpanded((prev) => !prev)}
      >
        <div className="flex items-center gap-2">
          <Upload className="h-4 w-4" />
          <span className="text-sm font-semibold">{t("upload.uploads")}</span>
          <Badge className="h-5 min-w-5 justify-center text-xs">
            {uploads.length}
          </Badge>
        </div>
        <ChevronDown
          className={`h-4 w-4 transition-transform ${isExpanded ? "" : "rotate-180"}`}
        />
      </button>

      {isExpanded && (
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
  );
};
