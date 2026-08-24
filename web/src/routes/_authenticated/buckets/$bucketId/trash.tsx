import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { TrashTab } from "@/components/bucket-view/components/TrashTab";
import { useTrashActions } from "@/components/bucket-view/hooks/useTrashActions";
import {
  bucketDataQueryOptions,
  bucketTrashedFilesQueryOptions,
} from "@/queries/bucket.ts";

export const Route = createFileRoute("/_authenticated/buckets/$bucketId/trash")(
  {
    loader: ({ context: { queryClient }, params: { bucketId } }) =>
      queryClient.ensureQueryData(bucketTrashedFilesQueryOptions(bucketId)),
    component: TrashRoute,
  },
);

function TrashRoute() {
  const { bucketId } = Route.useParams();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;
  const { trashedItems, restoreItem, purgeItem } = useTrashActions();

  return (
    <TrashTab
      items={trashedItems}
      bucket={bucket}
      onRestore={restoreItem}
      onPermanentDelete={purgeItem}
    />
  );
}
