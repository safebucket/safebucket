import React, { FC } from "react";

import { IBucket } from "@/components/bucket-view/helpers/types";
import { FileItemView } from "@/components/common/components/FileItemView";

interface IBucketGridViewProps {
  bucket: IBucket;
}

export const BucketGridView: FC<IBucketGridViewProps> = ({
  bucket,
}: IBucketGridViewProps) => {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
      {bucket.files.map((file) => (
        <FileItemView key={file.id} file={file} />
      ))}
    </div>
  );
};
