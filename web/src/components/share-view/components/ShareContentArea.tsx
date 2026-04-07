import { useTranslation } from "react-i18next";
import { ArrowLeft, FolderOpen } from "lucide-react";
import type { ColumnDef, VisibilityState } from "@tanstack/react-table";
import type { FC } from "react";

import type { BucketItem } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import type { ViewMode } from "@/components/share-view/components/ShareContentView.tsx";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { ShareListView } from "@/components/share-view/components/ShareListView.tsx";
import { ShareFileGridCard } from "@/components/share-view/components/ShareFileGridCard.tsx";

interface IShareContentAreaProps {
  items: Array<BucketItem>;
  columns: Array<ColumnDef<BucketItem>>;
  columnVisibility: VisibilityState;
  viewMode: ViewMode;
  folderName: string | null | undefined;
  canGoBack: boolean;
  onGoBack: () => void;
  onOpenFolder: (item: BucketItem) => void;
  onDownload: (file: IFile) => void;
}

export const ShareContentArea: FC<IShareContentAreaProps> = ({
  items,
  columns,
  columnVisibility,
  viewMode,
  folderName,
  canGoBack,
  onGoBack,
  onOpenFolder,
  onDownload,
}) => {
  const { t } = useTranslation();

  return (
    <>
      {canGoBack && (
        <button
          type="button"
          onClick={onGoBack}
          className="mb-4 flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          {folderName ?? t("share_consumer.back")}
        </button>
      )}

      {items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <FolderOpen className="text-muted-foreground mb-4 h-16 w-16" />
          <p className="text-muted-foreground text-lg">
            {t("share_consumer.empty")}
          </p>
        </div>
      ) : viewMode === "list" ? (
        <ShareListView
          columns={columns}
          data={items}
          onRowDoubleClick={onOpenFolder}
          defaultColumnVisibility={columnVisibility}
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
          {items.map((item) => (
            <ShareFileGridCard
              key={item.id}
              file={item}
              onDoubleClick={onOpenFolder}
              onDownload={!isFolder(item) ? () => onDownload(item) : undefined}
            />
          ))}
        </div>
      )}
    </>
  );
};
