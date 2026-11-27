import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Download, LoaderCircle, MessageSquare, XIcon } from "lucide-react";
import type { FC, ReactNode } from "react";

import type { IFile } from "@/types/file.ts";
import { PDFPreview } from "@/components/file-actions/components/PDFPreview";
import { getPreviewKind } from "@/components/file-actions/helpers/preview-kind";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn, formatFileSize } from "@/lib/utils";

const INLINE_PREVIEW_MAX_BYTES = 100 * 1024 * 1024; // 100 MiB
const CAN_STREAM = new Set(["video", "audio"]);
const DESKTOP_QUERY = "(min-width: 768px)";

interface IFilePreviewDialogProps {
  open: boolean;
  onOpenChange: (isOpen: boolean) => void;
  file: IFile;
  fetchUrl: () => Promise<{ url: string }>;
  onDownload: () => void;
  comments?: ReactNode;
  commentCount?: number;
}

export const FilePreviewDialog: FC<IFilePreviewDialogProps> = ({
  open,
  onOpenChange,
  file,
  fetchUrl,
  onDownload,
  comments,
  commentCount,
}: IFilePreviewDialogProps) => {
  const { t } = useTranslation();
  const [showComments, setShowComments] = useState(
    () => !!comments && window.matchMedia(DESKTOP_QUERY).matches,
  );
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
      <DialogContent
        showCloseButton={false}
        className={cn(
          "flex flex-col gap-0 overflow-hidden p-0",
          comments ? "h-[90dvh] sm:max-w-6xl" : "h-[75dvh] sm:max-w-4xl",
        )}
      >
        <div className="flex items-center gap-2 border-b px-6 py-4">
          <DialogHeader className="min-w-0 flex-1 gap-0.5">
            <DialogTitle className="truncate" title={file.name}>
              {file.name}
            </DialogTitle>
            <DialogDescription>{formatFileSize(file.size)}</DialogDescription>
          </DialogHeader>
          {comments && (
            <Button
              variant={showComments ? "secondary" : "ghost"}
              size="sm"
              aria-pressed={showComments}
              onClick={() => setShowComments(!showComments)}
            >
              <MessageSquare />
              {t("file_comments.title")}
              {!!commentCount && <Badge>{commentCount}</Badge>}
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("file_actions.download")}
            onClick={onDownload}
          >
            <Download />
          </Button>
          <DialogClose asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              className="bg-secondary"
              aria-label={t("common.close")}
            >
              <XIcon />
            </Button>
          </DialogClose>
        </div>
        <div className="flex min-h-0 flex-1">
          <div
            className={cn(
              "flex min-w-0 flex-1 items-center justify-center overflow-hidden bg-muted/30",
              showComments && "hidden md:flex",
            )}
          >
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
                className="max-h-full max-w-full object-contain p-4"
              />
            )}
            {url && kind === "video" && (
              // eslint-disable-next-line jsx-a11y/media-has-caption
              <video
                src={url}
                controls
                className="max-h-full max-w-full"
                preload="metadata"
              />
            )}
            {url && kind === "audio" && (
              // eslint-disable-next-line jsx-a11y/media-has-caption
              <audio src={url} controls className="w-full max-w-md" />
            )}
            {url && kind === "pdf" && <PDFPreview url={url} />}
            {url && kind === "text" && (
              <iframe
                src={url}
                title={file.name}
                sandbox=""
                className="h-full w-full border-0"
              />
            )}
          </div>
          {comments && showComments && (
            <aside className="flex w-full flex-col md:w-96 md:border-l">
              {comments}
            </aside>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
};
