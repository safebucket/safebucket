import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { bucketDataQueryOptions } from "@/queries/bucket.ts";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";

export const Route = createFileRoute("/_authenticated/buckets/$id/$")({
  loader: ({ context: { queryClient }, params: { id } }) =>
    queryClient.ensureQueryData(bucketDataQueryOptions(id)),
  component: BucketComponent,
});

function BucketComponent() {
  const { id, _splat } = Route.useParams();
  const bucketQuery = useSuspenseQuery(bucketDataQueryOptions(id));
  const bucket = bucketQuery.data;

  // The splat captures the folder ID if present, otherwise it's empty (root)
  const folderId = _splat && _splat.trim() !== "" ? _splat : null;

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
