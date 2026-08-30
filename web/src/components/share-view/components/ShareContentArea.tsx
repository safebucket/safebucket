import { ArrowLeft, FolderOpen } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import { ShareItemList } from "@/components/share-view/components/ShareItemList.tsx";
import { Button } from "@/components/ui/button.tsx";

interface IShareContentAreaProps {
  items: Array<BucketItem>;
  folderName: string | null | undefined;
  canGoBack: boolean;
  onGoBack: () => void;
  onOpenItem: (item: BucketItem) => void;
  onPreview: (file: IFile) => void;
  onDownload: (file: IFile) => void;
}

export const ShareContentArea: FC<IShareContentAreaProps> = ({
  items,
  folderName,
  canGoBack,
  onGoBack,
  onOpenItem,
  onPreview,
  onDownload,
}) => {
  const { t } = useTranslation();

  return (
    <>
      {canGoBack && (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="mb-3"
          onClick={onGoBack}
        >
          <ArrowLeft className="size-4" />
          {folderName ?? t("share_consumer.back")}
        </Button>
      )}

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <FolderOpen className="text-muted-foreground mb-4 size-16" />
          <p className="text-muted-foreground text-lg">
            {t("share_consumer.empty")}
          </p>
        </div>
      ) : (
        <ShareItemList
          items={items}
          onOpenItem={onOpenItem}
          onPreview={onPreview}
          onDownload={onDownload}
        />
      )}
    </>
  );
};
