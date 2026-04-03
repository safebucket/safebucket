import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView.tsx";
import { Card } from "@/components/ui/card.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { cn, formatDate, formatFileSize } from "@/lib/utils.ts";

interface IShareFileGridCardProps {
  file: BucketItem;
  onDoubleClick?: (item: BucketItem) => void;
}

export const ShareFileGridCard: FC<IShareFileGridCardProps> = ({
  file,
  onDoubleClick,
}) => {
  const { t } = useTranslation();
  const itemIsFolder = isFolder(file);

  return (
    <Card
      className={cn(
        "relative flex cursor-default flex-col gap-4 p-5 transition-all hover:shadow-md min-h-[180px]",
        itemIsFolder && "cursor-pointer",
      )}
      onDoubleClick={() => onDoubleClick?.(file)}
    >
      <div className="flex items-start gap-4">
        <div className="bg-muted flex aspect-square w-16 flex-shrink-0 items-center justify-center rounded-lg">
          <FileIconView
            className="h-8 w-8"
            isFolder={itemIsFolder}
            extension={!itemIsFolder ? file.extension : undefined}
          />
        </div>

        <div className="flex-1 min-w-0 flex flex-col gap-2">
          <h3
            className="font-medium text-sm leading-tight line-clamp-2"
            title={file.name}
          >
            {file.name}
          </h3>
          <p className="text-xs text-muted-foreground">
            {itemIsFolder
              ? t("share_consumer.type_folder")
              : formatFileSize(file.size)}
          </p>
        </div>
      </div>

      <div className="flex items-center justify-between gap-2 pt-2 mt-auto border-t">
        <Badge variant="secondary" className="text-xs">
          {itemIsFolder ? "folder" : file.extension}
        </Badge>
        <span
          className="text-xs text-muted-foreground"
          title={formatDate(file.created_at)}
        >
          {formatDate(file.created_at)}
        </span>
      </div>
    </Card>
  );
};
