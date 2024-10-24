import React, { FC } from "react";

import { IFile } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FileItemView } from "@/components/common/components/FileItemView";

interface IBucketGridViewProps {
  files: IFile[];
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  files,
}: IBucketGridViewProps) => {
  const { selected, setSelected, openFolder } = useBucketViewContext();

  return (
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
  );
};
