import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { bucketDataQueryOptions } from "@/queries/bucket.ts";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/{-$folderId}",
)({
  loader: ({ context: { queryClient }, params: { bucketId } }) =>
    queryClient.ensureQueryData(bucketDataQueryOptions(bucketId)),
  component: BucketComponent,
});

function BucketComponent() {
  const { bucketId, folderId } = Route.useParams();
  const bucketQuery = useSuspenseQuery(bucketDataQueryOptions(bucketId));
  const bucket = bucketQuery.data;

  return (
    <BucketViewProvider folderId={folderId}>
      <div className="w-full flex-1">
        <div className="m-6 mt-0 grid grid-cols-1 gap-8">
          <BucketView bucket={bucket} />
        </div>
      </div>
    </BucketViewProvider>
  );
}
