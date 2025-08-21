import React, { FC } from "react";

import { FolderOpen } from "lucide-react";

import { IFile } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileItemView } from "@/components/common/components/FileItemView";
import { DragDropZone } from "@/components/upload/components/DragDropZone";

interface IBucketGridViewProps {
  files: IFile[];
  bucketId: string;
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  files,
  bucketId,
}: IBucketGridViewProps) => {
  const { selected, setSelected, openFolder } = useBucketViewContext();

  if (files.length === 0) {
    return (
      <DragDropZone bucketId={bucketId} className="min-h-[400px]">
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <FolderOpen className="h-16 w-16 text-muted-foreground mb-4" />
          <h3 className="text-lg font-semibold text-muted-foreground mb-2">
            This folder is empty
          </h3>
          <p className="text-sm text-muted-foreground max-w-sm">
            Upload files or create folders to get started organizing your content.
          </p>
          <p className="text-xs text-muted-foreground mt-2">
            You can also drag and drop files here to upload them.
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
