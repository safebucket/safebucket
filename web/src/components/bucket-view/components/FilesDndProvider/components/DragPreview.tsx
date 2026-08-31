import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { BucketItem } from "@/types/bucket.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { isFolder } from "@/components/bucket-view/helpers/utils";

interface IDragPreviewProps {
  items: Array<BucketItem>;
}

export const DragPreview: FC<IDragPreviewProps> = ({ items }) => {
  const { t } = useTranslation();

  if (items.length === 0) return null;

  const visible = items.slice(0, 3);
  const hiddenCount = Math.max(0, items.length - 3);

  return (
    <div className="divide-border bg-card w-72 divide-y overflow-hidden rounded-xl border shadow-lg">
      {visible.map((item) => (
        <div key={item.id} className="flex items-center gap-2.5 px-4 py-3">
          <FileIconView
            className="text-primary h-5 w-5 shrink-0"
            isFolder={isFolder(item)}
            extension={!isFolder(item) ? item.extension : undefined}
          />
          <span className="truncate text-sm font-medium">{item.name}</span>
        </div>
      ))}
      {hiddenCount > 0 ? (
        <div className="text-muted-foreground px-4 py-1.5 text-xs">
          {t("bucket.view.dragging_more", { count: hiddenCount })}
        </div>
      ) : null}
    </div>
  );
};
