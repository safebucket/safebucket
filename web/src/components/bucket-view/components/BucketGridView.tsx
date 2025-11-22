import { FolderOpen } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileGridCard } from "@/components/bucket-view/components/FileGridCard";
import { DragDropZone } from "@/components/upload/components/DragDropZone";
import type { BucketItem } from "@/types/bucket.ts";

interface IBucketGridViewProps {
  items: Array<BucketItem>;
  bucketId: string;
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  items,
  bucketId,
}: IBucketGridViewProps) => {
  const { t } = useTranslation();
  const { selected, setSelected, openFolder } = useBucketViewContext();

  if (items.length === 0) {
    return (
      <DragDropZone bucketId={bucketId} className="min-h-[400px]">
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <FolderOpen className="text-muted-foreground mb-4 h-16 w-16" />
          <h3 className="text-muted-foreground mb-2 text-lg font-semibold">
            {t("bucket.grid_view.empty_folder")}
          </h3>
          <p className="text-muted-foreground max-w-sm text-sm">
            {t("bucket.grid_view.empty_description")}
          </p>
          <p className="text-muted-foreground mt-2 text-xs">
            {t("bucket.grid_view.drag_drop_hint")}
          </p>
        </div>
      </DragDropZone>
    );
  }

  return (
    <DragDropZone bucketId={bucketId}>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
        {items.map((item) => (
          <FileGridCard
            key={item.id}
            file={item}
            selected={selected}
            setSelected={setSelected}
            onDoubleClick={openFolder}
          />
        ))}
      </div>
    </DragDropZone>
  );
};
