import { Check, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IUpload } from "@/components/upload/helpers/types";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { getFileExtension } from "@/components/upload/helpers/file-processing";
import { resolveErrorMessage } from "@/lib/toast";
import { formatFileSize } from "@/lib/utils";

interface IUploadRowProps {
  upload: IUpload;
  isComplete: boolean;
  onCancel: (uploadId: string) => void;
}

export const UploadRow: FC<IUploadRowProps> = ({
  upload,
  isComplete,
  onCancel,
}) => {
  const { t } = useTranslation();
  const { id, status, name, progress, size, error } = upload;

  const isActive = status === "uploading" || status === "queued";
  const showProgress =
    !isComplete && status !== "error" && status !== "cancelled";

  return (
    <div className="px-4 py-3">
      <div className="flex items-center gap-3">
        <FileIconView
          isFolder={false}
          extension={getFileExtension(name)}
          className="text-muted-foreground size-5 shrink-0"
        />
        <p
          className="min-w-0 flex-1 truncate text-sm font-medium"
          title={upload.path}
        >
          {name}
        </p>

        {status === "success" && (
          <Check className="text-success size-5 shrink-0" />
        )}

        {isActive && (
          <div className="flex shrink-0 items-center gap-1">
            <span className="text-muted-foreground text-xs whitespace-nowrap">
              {status === "queued"
                ? t("upload.status.waiting")
                : `${progress}%`}
            </span>
            <Button
              variant="ghost"
              size="icon-xs"
              className="text-muted-foreground"
              onClick={() => onCancel(id)}
              aria-label={t("upload.cancel_file", { name })}
            >
              <X />
            </Button>
          </div>
        )}
      </div>

      {showProgress && (
        <Progress
          value={status === "success" ? 100 : progress}
          className="mt-2 h-1.5"
          indicatorClassName={
            status === "success" ? "bg-success" : "bg-primary"
          }
        />
      )}

      {status === "error" && (
        <p className="text-destructive mt-1 truncate text-xs">
          {t("upload.status.failed")}
          {error ? ` · ${resolveErrorMessage(error)}` : ""}
        </p>
      )}

      {status === "cancelled" && (
        <p className="text-muted-foreground mt-1 truncate text-xs">
          {t("upload.status.cancelled")}
        </p>
      )}

      {isComplete && status === "success" && (
        <p className="text-muted-foreground mt-1 truncate text-xs">
          {size != null ? `${formatFileSize(size)} · ` : ""}
          {t("upload.status.uploaded")}
        </p>
      )}
    </div>
  );
};
