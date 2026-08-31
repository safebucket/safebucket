import { useMemo } from "react";
import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { FilePreviewMount } from "@/components/bucket-view/components/FilePreviewMount.tsx";
import { FilesTab } from "@/components/bucket-view/components/FilesTab/FilesTab";
import { itemsToShow } from "@/components/bucket-view/helpers/utils";
import { useBucketPermissions } from "@/hooks/usePermissions";
import { bucketDataQueryOptions } from "@/queries/bucket.ts";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/files/{-$folderId}",
)({
  validateSearch: (search: Record<string, unknown>): { preview?: string } => ({
    preview: typeof search.preview === "string" ? search.preview : undefined,
  }),
  component: FilesRoute,
});

function FilesRoute() {
  const { bucketId, folderId } = Route.useParams();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;
  const { isContributor } = useBucketPermissions(bucketId);

  const items = useMemo(
    () => itemsToShow(bucket.files, bucket.folders, folderId),
    [bucket, folderId],
  );

  return (
    <>
      <FilesTab
        bucket={bucket}
        items={items}
        folderId={folderId}
        isContributor={isContributor}
      />
      <FilePreviewMount />
    </>
  );
}
