import { ChevronRight, Download, Eye } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { formatDate, formatFileSize } from "@/lib/utils.ts";

interface IShareItemListProps {
  items: Array<BucketItem>;
  onOpenItem: (item: BucketItem) => void;
  onPreview: (file: IFile) => void;
  onDownload: (file: IFile) => void;
}

export const ShareItemList: FC<IShareItemListProps> = ({
  items,
  onOpenItem,
  onPreview,
  onDownload,
}) => {
  const { t } = useTranslation();

  return (
    <div className="min-w-0 divide-y">
      {items.map((item) => {
        const itemIsFolder = isFolder(item);

        return (
          <div key={item.id} className="flex min-w-0 items-center gap-3 py-3">
            <div className="bg-secondary flex size-11 shrink-0 items-center justify-center rounded-xl">
              <FileIconView
                className="size-5 text-primary"
                isFolder={itemIsFolder}
                extension={!itemIsFolder ? item.extension : undefined}
              />
            </div>

            <div className="min-w-0 flex-1">
              <p className="truncate font-medium" title={item.name}>
                {item.name}
              </p>
              <p className="text-muted-foreground truncate text-sm">
                {itemIsFolder
                  ? `${t("share_consumer.type_folder")} · ${formatDate(item.created_at)}`
                  : `${formatFileSize(item.size)} · ${formatDate(item.created_at)}`}
              </p>
            </div>

            <Badge
              variant="secondary"
              className="hidden shrink-0 sm:inline-flex"
            >
              {itemIsFolder ? t("share_consumer.type_folder") : item.extension}
            </Badge>

            {itemIsFolder ? (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => onOpenItem(item)}
                title={t("share_consumer.open_folder")}
              >
                <ChevronRight className="size-4" />
              </Button>
            ) : (
              <div className="flex items-center">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => onPreview(item)}
                  title={t("file_actions.preview")}
                >
                  <Eye className="size-4" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => onDownload(item)}
                  title={t("common.download")}
                >
                  <Download className="size-4" />
                </Button>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
