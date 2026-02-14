import { useEffect, useState } from "react";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { BucketActivityView } from "@/components/bucket-view/components/BucketActivityView";
import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { BucketHeader } from "@/components/bucket-view/components/BucketHeader";
import { BucketListView } from "@/components/bucket-view/components/BucketListView";
import { BucketSettings } from "@/components/bucket-view/components/BucketSettings";
import { BucketTrashView } from "@/components/bucket-view/components/BucketTrashView";
import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { itemsToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useTrashActions } from "@/components/bucket-view/hooks/useTrashActions";
import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { useUploadDialog } from "@/components/upload/hooks/useUploadDialog";

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
  const uploadDialog = useUploadDialog({ bucketId: bucket.id, folderId });

  useEffect(() => {
    setItems(itemsToShow(bucket.files, bucket.folders, folderId));
  }, [bucket, folderId]);

  const viewComponents = {
    [BucketViewMode.List]: (
      <BucketListView
        items={items}
        bucketId={bucket.id}
        onFilesDropped={uploadDialog.openWithFiles}
      />
    ),
    [BucketViewMode.Grid]: (
      <BucketGridView
        items={items}
        bucketId={bucket.id}
        onFilesDropped={uploadDialog.openWithFiles}
      />
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
      <BucketHeader bucket={bucket} onOpenUploadDialog={uploadDialog.open} />

      {viewComponents[view]}

      <UploadDialog
        {...uploadDialog.dialogProps}
        stagedFiles={uploadDialog.stagedFiles}
        onAddFiles={uploadDialog.addFiles}
        onRemoveFile={uploadDialog.removeFile}
        expiresAt={uploadDialog.expiresAt}
        onExpiresAtChange={uploadDialog.setExpiresAt}
        isAdvancedOpen={uploadDialog.isAdvancedOpen}
        onAdvancedOpenChange={uploadDialog.setIsAdvancedOpen}
        onUpload={uploadDialog.handleUpload}
      />
    </>
  );
};
