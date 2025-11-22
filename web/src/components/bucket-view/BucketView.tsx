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
import { itemsToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useTrashActions } from "@/components/bucket-view/hooks/useTrashActions";

interface IBucketViewProps {
  bucket: IBucket;
}

export const BucketView: FC<IBucketViewProps> = ({
  bucket,
}: IBucketViewProps) => {
  const { folderId, view } = useBucketViewContext();
  const [items, setItems] = useState(
    itemsToShow(bucket.files, bucket.folders, folderId),
  );
  const { trashedItems, restoreItem, purgeItem } = useTrashActions();

  useEffect(() => {
    setItems(itemsToShow(bucket.files, bucket.folders, folderId));
  }, [bucket, folderId]);

  const viewComponents = {
    [BucketViewMode.List]: (
      <BucketListView items={items} bucketId={bucket.id} />
    ),
    [BucketViewMode.Grid]: (
      <BucketGridView items={items} bucketId={bucket.id} />
    ),
    [BucketViewMode.Activity]: <BucketActivityView />,
    [BucketViewMode.Trash]: (
      <BucketTrashView
        items={trashedItems}
        bucket={bucket}
        onRestore={restoreItem}
        onPermanentDelete={purgeItem}
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
