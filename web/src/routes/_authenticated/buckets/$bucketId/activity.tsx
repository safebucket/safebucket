import { createFileRoute } from "@tanstack/react-router";

import { ActivityTab } from "@/components/bucket-view/components/ActivityTab";
import { bucketActivityInfiniteQueryOptions } from "@/queries/bucket.ts";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/activity",
)({
  loader: ({ context: { queryClient }, params: { bucketId } }) =>
    queryClient.ensureInfiniteQueryData(
      bucketActivityInfiniteQueryOptions(bucketId),
    ),
  component: ActivityTab,
});
