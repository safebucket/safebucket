import { useState } from "react";

import { ChevronDown, Upload, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { resolveErrorMessage } from "@/components/ui/hooks/use-toast";
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
    <div className="fixed inset-x-4 bottom-8 z-50 mx-auto max-w-96 rounded-lg border bg-card text-card-foreground shadow-lg md:inset-x-auto md:right-8 md:mx-0 md:w-96 md:max-w-lg">
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

          <div className="mt-2 flex max-h-64 flex-col gap-2 overflow-y-auto md:max-h-96">
            {uploads.map((upload) => {
              const state =
                upload.status === "success"
                  ? "done"
                  : upload.status === "error"
                    ? "error"
                    : upload.progress === 0
                      ? "processing"
                      : "uploading";

              return (
                <Attachment
                  key={upload.id}
                  size="sm"
                  state={state}
                  className="w-full"
                >
                  <AttachmentMedia>
                    {getStatusIcon(upload.status, upload.progress)}
                  </AttachmentMedia>
                  <AttachmentContent>
                    <AttachmentTitle title={upload.path}>
                      {upload.name}
                    </AttachmentTitle>
                    {upload.status === "uploading" ? (
                      <div className="mt-1 flex items-center gap-2">
                        <Progress
                          value={upload.progress}
                          className="h-1.5 flex-1"
                        />
                        <span className="text-muted-foreground text-xs whitespace-nowrap">
                          {getStatusText(upload.status, upload.progress, t)}
                        </span>
                      </div>
                    ) : (
                      <AttachmentDescription>
                        {upload.status === "error" && upload.error
                          ? resolveErrorMessage(upload.error)
                          : getStatusText(upload.status, upload.progress, t)}
                      </AttachmentDescription>
                    )}
                  </AttachmentContent>
                  {upload.status === "uploading" && (
                    <AttachmentActions>
                      <AttachmentAction
                        onClick={() => cancelUpload(upload.id)}
                        aria-label={`Cancel upload: ${upload.name}`}
                      >
                        <X />
                      </AttachmentAction>
                    </AttachmentActions>
                  )}
                </Attachment>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
