import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Maximize2, XIcon } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file";
import { useOpenBucketItem } from "@/components/bucket-view/hooks/useOpenBucketItem";
import { FileCommentsPanel } from "@/components/file-comments/FileCommentsPanel";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useIsMobile } from "@/hooks/use-mobile";
import { formatFileSize } from "@/lib/utils";
import { fileCommentsQueryOptions } from "@/queries/file-comments";

interface IFileCommentsSheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  bucketId: string;
  file: IFile;
}

export const FileCommentsSheet: FC<IFileCommentsSheetProps> = ({
  open,
  onOpenChange,
  bucketId,
  file,
}: IFileCommentsSheetProps) => {
  const { t } = useTranslation();
  const openItem = useOpenBucketItem();
  const isMobile = useIsMobile();
  const comments = useQuery({
    ...fileCommentsQueryOptions(bucketId, file.id),
    enabled: open,
  });

  const openPreview = () => {
    onOpenChange(false);
    openItem(file);
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={isMobile ? "bottom" : "right"}
        showCloseButton={false}
        className="gap-0 data-[side=bottom]:h-[85dvh] data-[side=right]:w-full data-[side=right]:sm:max-w-md"
      >
        <SheetHeader className="gap-3 border-b p-4">
          <div className="flex items-start gap-2">
            <div className="min-w-0 flex-1">
              <SheetTitle className="truncate" title={file.name}>
                {file.name}
              </SheetTitle>
              <SheetDescription>
                {formatFileSize(file.size)}
                {comments.isSuccess &&
                  ` · ${t("file_comments.count", { count: comments.data.length })}`}
              </SheetDescription>
            </div>
            <SheetClose asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                className="bg-secondary"
                aria-label={t("common.close")}
              >
                <XIcon />
              </Button>
            </SheetClose>
          </div>
          <Button variant="outline" onClick={openPreview}>
            <Maximize2 />
            {t("file_comments.open_preview")}
          </Button>
        </SheetHeader>
        <FileCommentsPanel bucketId={bucketId} fileId={file.id} />
      </SheetContent>
    </Sheet>
  );
};
