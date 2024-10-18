import React, { FC, useState } from "react";

import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { BucketHeader } from "@/components/bucket-view/components/BucketHeader";
import { BucketListView } from "@/components/bucket-view/components/BucketListView";
import {
  BucketViewMode,
  IBucket,
} from "@/components/bucket-view/helpers/types";

interface IBucketViewProps {
  bucket: IBucket;
}

export const BucketView: FC<IBucketViewProps> = ({
  bucket,
}: IBucketViewProps) => {
  const [view, setView] = useState<BucketViewMode>(BucketViewMode.List);

  return (
    <>
      <BucketHeader view={view} setView={setView} bucket={bucket} />

      {view == BucketViewMode.List ? (
        <BucketListView bucket={bucket} />
      ) : (
        <BucketGridView bucket={bucket} />
      )}
    </>
  );
};
