import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { bucketDataQueryOptions } from "@/queries/bucket.ts";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";

export const Route = createFileRoute("/buckets/$id/{-$path}")({
  loader: ({ context: { queryClient }, params: { id } }) =>
    queryClient.ensureQueryData(bucketDataQueryOptions(id)),
  component: BucketComponent,
});

function BucketComponent() {
  const { id, path } = Route.useParams();
  const bucketQuery = useSuspenseQuery(bucketDataQueryOptions(id));
  const bucket = bucketQuery.data;

  const joinedPath = path ? `/${path}` : "/";

  return (
    <BucketViewProvider path={joinedPath}>
      <div className="w-full flex-1">
        <div className="m-6 mt-0 grid grid-cols-1 gap-8">
          <BucketView bucket={bucket} />
        </div>
      </div>
    </BucketViewProvider>
  );
}
