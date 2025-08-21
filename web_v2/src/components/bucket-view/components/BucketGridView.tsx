import { FolderOpen } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";

import type { IFile } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileItemView } from "@/components/common/components/FileItemView";
import { DragDropZone } from "@/components/upload/components/DragDropZone";

interface IBucketGridViewProps {
  files: Array<IFile>;
  bucketId: string;
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  files,
  bucketId,
}: IBucketGridViewProps) => {
  const { t } = useTranslation();
  const { selected, setSelected, openFolder } = useBucketViewContext();

  if (files.length === 0) {
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
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
        {files.map((file) => (
          <FileItemView
            key={file.id}
            file={file}
            selected={selected}
            setSelected={setSelected}
            onDoubleClick={openFolder}
          />
        ))}
      </div>
    </DragDropZone>
  );
};
