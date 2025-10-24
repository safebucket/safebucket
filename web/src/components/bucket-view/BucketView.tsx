import { useEffect, useState } from "react";
import type { FC } from "react";

import type { IBucket } from "@/types/bucket.ts";
import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { BucketActivityView } from "@/components/bucket-view/components/BucketActivityView";
import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { BucketHeader } from "@/components/bucket-view/components/BucketHeader";
import { BucketListView } from "@/components/bucket-view/components/BucketListView";
import { BucketSettings } from "@/components/bucket-view/components/BucketSettings";
import { BucketTrashView } from "@/components/bucket-view/components/BucketTrashView";
import { filesToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useTrashActions } from "@/components/bucket-view/hooks/useTrashActions";

interface IBucketViewProps {
  bucket: IBucket;
}

export const BucketView: FC<IBucketViewProps> = ({
  bucket,
}: IBucketViewProps) => {
  const { path, view } = useBucketViewContext();
  const [files, setFiles] = useState(filesToShow(bucket.files, path));
  const { trashedFiles, restoreFile, purgeFile } = useTrashActions();

  useEffect(() => {
    setFiles(filesToShow(bucket.files, path));
  }, [bucket, path]);

  const viewComponents = {
    [BucketViewMode.List]: (
      <BucketListView files={files} bucketId={bucket.id} />
    ),
    [BucketViewMode.Grid]: (
      <BucketGridView files={files} bucketId={bucket.id} />
    ),
    [BucketViewMode.Activity]: <BucketActivityView />,
    [BucketViewMode.Trash]: (
      <BucketTrashView
        files={trashedFiles}
        onRestore={restoreFile}
        onPermanentDelete={purgeFile}
      />
    ),
    [BucketViewMode.Settings]: <BucketSettings bucket={bucket} />,
  };

  return (
    <>
      <BucketHeader bucket={bucket} />

      {viewComponents[view]}
    </>
  );
};
