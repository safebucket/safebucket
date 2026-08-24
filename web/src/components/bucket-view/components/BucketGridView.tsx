import { FolderOpen } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { BucketItem } from "@/types/bucket.ts";
import { FileGridCard } from "@/components/bucket-view/components/FileGridCard";
import { useOpenBucketItem } from "@/components/bucket-view/hooks/useOpenBucketItem";

interface IBucketGridViewProps {
  items: Array<BucketItem>;
  selected: BucketItem | null;
  setSelected: (item: BucketItem) => void;
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  items,
  selected,
  setSelected,
}: IBucketGridViewProps) => {
  const { t } = useTranslation();
  const openItem = useOpenBucketItem();

  if (items.length === 0) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center py-12 text-center">
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
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
      {items.map((item) => (
        <FileGridCard
          key={item.id}
          file={item}
          selected={selected}
          setSelected={setSelected}
          onDoubleClick={openItem}
        />
      ))}
    </div>
  );
};
