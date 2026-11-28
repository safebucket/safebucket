import { useQuery, useSuspenseQuery } from "@tanstack/react-query";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import type { FC } from "react";

import { FilePreviewDialog } from "@/components/file-actions/components/FilePreviewDialog";
import { api_downloadFile } from "@/components/file-actions/helpers/api";
import { useFileActions } from "@/components/file-actions/hooks/useFileActions";
import { FileCommentsPanel } from "@/components/file-comments/FileCommentsPanel";
import { bucketDataQueryOptions } from "@/queries/bucket.ts";
import { fileCommentsQueryOptions } from "@/queries/file-comments";

export const FilePreviewMount: FC = () => {
  const { bucketId, folderId } = useParams({
    from: "/_authenticated/buckets/$bucketId/files/{-$folderId}",
  });
  const { preview } = useSearch({
    from: "/_authenticated/buckets/$bucketId/files/{-$folderId}",
  });
  const navigate = useNavigate();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;
  const { downloadFile } = useFileActions();
  const comments = useQuery({
    ...fileCommentsQueryOptions(bucketId, preview ?? ""),
    enabled: !!preview,
  });

  const file = preview
    ? bucket.files.find((item) => item.id === preview)
    : undefined;

  if (!file) return null;

  const close = () =>
    navigate({
      to: "/buckets/$bucketId/files/{-$folderId}",
      params: { bucketId, folderId },
      search: {},
    });

  return (
    <FilePreviewDialog
      open
      onOpenChange={(isOpen) => {
        if (!isOpen) close();
      }}
      file={file}
      fetchUrl={() => api_downloadFile(bucketId, file.id, "preview")}
      onDownload={() => {
        close();
        downloadFile(file.id, file.name);
      }}
      comments={<FileCommentsPanel bucketId={bucketId} fileId={file.id} />}
      commentCount={comments.data?.length}
    />
  );
};
