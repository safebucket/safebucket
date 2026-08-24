import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { MembersTab } from "@/components/bucket-view/components/MembersTab";
import {
  bucketDataQueryOptions,
  bucketMembersQueryOptions,
} from "@/queries/bucket.ts";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/members",
)({
  loader: ({ context: { queryClient }, params: { bucketId } }) =>
    queryClient.ensureQueryData(bucketMembersQueryOptions(bucketId)),
  component: MembersRoute,
});

function MembersRoute() {
  const { bucketId } = Route.useParams();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;

  return <MembersTab bucket={bucket} />;
}
