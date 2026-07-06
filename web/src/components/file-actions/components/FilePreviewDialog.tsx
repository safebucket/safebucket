import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Download, LoaderCircle } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file.ts";
import { getPreviewKind } from "@/components/file-actions/helpers/preview-kind";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

const INLINE_PREVIEW_MAX_BYTES = 100 * 1024 * 1024; // 100 MiB
const CAN_STREAM = new Set(["video", "audio"]);

interface IFilePreviewDialogProps {
  open: boolean;
  onOpenChange: (isOpen: boolean) => void;
  file: IFile;
  fetchUrl: () => Promise<{ url: string }>;
  onDownload: () => void;
}

export const FilePreviewDialog: FC<IFilePreviewDialogProps> = ({
  open,
  onOpenChange,
  file,
  fetchUrl,
  onDownload,
}: IFilePreviewDialogProps) => {
  const { t } = useTranslation();
  const kind = getPreviewKind(file.extension);
  const tooLarge =
    !CAN_STREAM.has(kind) && file.size > INLINE_PREVIEW_MAX_BYTES;
  const canPreview = kind !== "unsupported" && !tooLarge;

  const { data, isLoading, isError } = useQuery({
    queryKey: [file.id, "preview"],
    queryFn: fetchUrl,
    enabled: open && canPreview,
    staleTime: 0,
    gcTime: 0,
  });

  const url = data?.url;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle className="truncate pr-8" title={file.name}>
            {file.name}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {t("file_actions.preview")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex min-h-[60vh] items-center justify-center overflow-hidden bg-muted/30">
          {kind === "unsupported" && (
            <div className="flex flex-col items-center gap-3 p-6 text-center">
              <p className="text-sm text-muted-foreground">
                {t("file_actions.preview_unsupported")}
              </p>
              <Button onClick={onDownload}>
                <Download className="mr-2 h-4 w-4" />
                {t("file_actions.download")}
              </Button>
            </div>
          )}
          {tooLarge && kind !== "unsupported" && (
            <div className="flex flex-col items-center gap-3 p-6 text-center">
              <p className="text-sm text-muted-foreground">
                {t("file_actions.preview_too_large")}
              </p>
              <Button onClick={onDownload}>
                <Download className="mr-2 h-4 w-4" />
                {t("file_actions.download")}
              </Button>
            </div>
          )}
          {canPreview && isLoading && (
            <LoaderCircle className="h-8 w-8 animate-spin text-muted-foreground" />
          )}
          {canPreview && isError && (
            <p className="p-6 text-sm text-destructive">
              {t("file_actions.preview_failed")}
            </p>
          )}
          {url && kind === "image" && (
            <img
              src={url}
              alt={file.name}
              className="max-h-[70vh] max-w-full object-contain"
            />
          )}
          {url && kind === "video" && (
            // eslint-disable-next-line jsx-a11y/media-has-caption
            <video
              src={url}
              controls
              className="max-h-[70vh] max-w-full"
              preload="metadata"
            />
          )}
          {url && kind === "audio" && (
            // eslint-disable-next-line jsx-a11y/media-has-caption
            <audio src={url} controls className="w-full max-w-md" />
          )}
          {url && (kind === "pdf" || kind === "text") && (
            <iframe
              src={url}
              title={file.name}
              sandbox=""
              className="h-[70vh] w-full border-0"
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
