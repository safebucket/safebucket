import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { ShareLinksTab } from "@/components/bucket-view/components/ShareLinksTab";
import {
  bucketDataQueryOptions,
  bucketSharesQueryOptions,
} from "@/queries/bucket.ts";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/shares",
)({
  loader: ({ context: { queryClient }, params: { bucketId } }) =>
    queryClient.ensureQueryData(bucketSharesQueryOptions(bucketId)),
  component: SharesRoute,
});

function SharesRoute() {
  const { bucketId } = Route.useParams();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;

  return <ShareLinksTab bucket={bucket} />;
}
