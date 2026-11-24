import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { Maximize2, XIcon } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file";
import { useOpenBucketItem } from "@/components/bucket-view/hooks/useOpenBucketItem";
import { FileCommentsPanel } from "@/components/file-comments/FileCommentsPanel";
import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { formatFileSize } from "@/lib/utils";
import { fileCommentsQueryOptions } from "@/queries/file-comments";

interface IFileCommentsDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  bucketId: string;
  file: IFile;
}

export const FileCommentsDrawer: FC<IFileCommentsDrawerProps> = ({
  open,
  onOpenChange,
  bucketId,
  file,
}: IFileCommentsDrawerProps) => {
  const { t } = useTranslation();
  const openItem = useOpenBucketItem();
  const comments = useQuery(fileCommentsQueryOptions(bucketId, file.id));
  const direction = window.matchMedia("(min-width: 768px)").matches
    ? "right"
    : "bottom";

  const openPreview = () => {
    onOpenChange(false);
    openItem(file);
  };

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction={direction}>
      <DrawerContent className="p-2 data-[vaul-drawer-direction=bottom]:h-[85dvh] data-[vaul-drawer-direction=bottom]:max-h-[85dvh] data-[vaul-drawer-direction=right]:w-full data-[vaul-drawer-direction=right]:sm:max-w-md">
        <DrawerHeader className="gap-3 border-b p-4 text-left md:gap-3">
          <div className="flex items-start gap-2">
            <div className="min-w-0 flex-1">
              <DrawerTitle className="truncate" title={file.name}>
                {file.name}
              </DrawerTitle>
              <DrawerDescription>
                {formatFileSize(file.size)}
                {comments.isSuccess &&
                  ` · ${t("file_comments.count", { count: comments.data.length })}`}
              </DrawerDescription>
            </div>
            <DrawerClose asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                className="bg-secondary"
                aria-label={t("common.close")}
              >
                <XIcon />
              </Button>
            </DrawerClose>
          </div>
          <Button variant="outline" onClick={openPreview}>
            <Maximize2 />
            {t("file_comments.open_preview")}
          </Button>
        </DrawerHeader>
        <FileCommentsPanel bucketId={bucketId} fileId={file.id} />
      </DrawerContent>
    </Drawer>
  );
};
