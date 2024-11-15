import React, { FC, useEffect, useState } from "react";

import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { BucketHeader } from "@/components/bucket-view/components/BucketHeader";
import { BucketListView } from "@/components/bucket-view/components/BucketListView";
import {
  BucketViewMode,
  IBucket,
} from "@/components/bucket-view/helpers/types";
import { filesToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";

interface IBucketViewProps {
  bucket: IBucket;
}

export const BucketView: FC<IBucketViewProps> = ({
  bucket,
}: IBucketViewProps) => {
  const { path, view } = useBucketViewContext();
  const [files, setFiles] = useState(filesToShow(bucket.files, path));

  useEffect(() => {
    setFiles(filesToShow(bucket.files, path));
  }, [bucket, path]);

  return (
    <>
      <BucketHeader bucket={bucket} />

      {view == BucketViewMode.List ? (
        <BucketListView files={files} />
      ) : (
        <BucketGridView files={files} />
      )}
    </>
  );
};
