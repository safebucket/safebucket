import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Download, LoaderCircle } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file.ts";
import { api_downloadFile } from "@/components/file-actions/helpers/api";
import {
  getPreviewKind,
  getPreviewMime,
} from "@/components/file-actions/helpers/preview-kind";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";

interface IFilePreviewDialogProps {
  open: boolean;
  onOpenChange: (isOpen: boolean) => void;
  bucketId: string;
  file: IFile;
  onDownload: () => void;
}

export const FilePreviewDialog: FC<IFilePreviewDialogProps> = ({
  open,
  onOpenChange,
  bucketId,
  file,
  onDownload,
}: IFilePreviewDialogProps) => {
  const { t } = useTranslation();
  const kind = getPreviewKind(file.extension);

  const { data, isLoading, isError } = useQuery({
    queryKey: [bucketId, file.id, "preview"],
    queryFn: () => api_downloadFile(bucketId, file.id),
    enabled: open && kind !== "unsupported",
    staleTime: 0,
    gcTime: 0,
  });

  const url = data?.url;
  const needsBlob = kind === "pdf" || kind === "text";
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [blobError, setBlobError] = useState(false);

  useEffect(() => {
    if (!url || !needsBlob) {
      setBlobUrl(null);
      setBlobError(false);
      return;
    }

    let cancelled = false;
    let createdUrl: string | null = null;

    fetch(url)
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.blob();
      })
      .then((blob) => {
        if (cancelled) return;
        const typed = new Blob([blob], {
          type: getPreviewMime(kind, file.extension),
        });
        createdUrl = URL.createObjectURL(typed);
        setBlobUrl(createdUrl);
      })
      .catch(() => {
        if (!cancelled) setBlobError(true);
      });

    return () => {
      cancelled = true;
      if (createdUrl) URL.revokeObjectURL(createdUrl);
    };
  }, [url, needsBlob, kind, file.extension]);

  const iframeSrc = needsBlob ? blobUrl : url;
  const showLoader =
    kind !== "unsupported" &&
    (isLoading || (needsBlob && !blobUrl && !blobError));
  const showError = isError || blobError;

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
          {showLoader && (
            <LoaderCircle className="h-8 w-8 animate-spin text-muted-foreground" />
          )}
          {showError && !showLoader && (
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
          {iframeSrc && (kind === "pdf" || kind === "text") && (
            <iframe
              src={iframeSrc}
              title={file.name}
              className="h-[70vh] w-full border-0"
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
