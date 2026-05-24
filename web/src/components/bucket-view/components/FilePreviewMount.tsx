import type { FC } from "react";

import { isFile } from "@/components/bucket-view/helpers/utils";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FilePreviewDialog } from "@/components/file-actions/components/FilePreviewDialog";
import { api_downloadFile } from "@/components/file-actions/helpers/api";
import { useFileActions } from "@/components/file-actions/hooks/useFileActions";

export const FilePreviewMount: FC = () => {
  const { bucketId, previewItem, closePreview } = useBucketViewContext();
  const { downloadFile } = useFileActions();

  if (!previewItem || !isFile(previewItem)) return null;

  return (
    <FilePreviewDialog
      open
      onOpenChange={(isOpen) => {
        if (!isOpen) closePreview();
      }}
      file={previewItem}
      fetchUrl={() => api_downloadFile(bucketId, previewItem.id, "preview")}
      onDownload={() => {
        closePreview();
        downloadFile(previewItem.id, previewItem.name);
      }}
    />
  );
};
