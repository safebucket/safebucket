import { createFileRoute } from "@tanstack/react-router";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketSkeleton } from "@/components/bucket-view/components/BucketSkeleton";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";
import { useBucketData } from "@/components/bucket-view/hooks/useBucketData";

export const Route = createFileRoute("/buckets/$id/$")({
  component: BucketComponent,
});

function BucketComponent() {
  const { id, _splat } = Route.useParams();
  const { bucket, isLoading } = useBucketData(id);

  // Convert splat parameter to path, similar to Next.js [...path] handling
  const path = _splat ? `/${_splat}` : "/";

  return (
    <BucketViewProvider path={path}>
      <div className="flex-1 w-full">
        <div className="m-6 mt-0 grid grid-cols-1 gap-8">
          {isLoading ? (
            <BucketSkeleton />
          ) : (
            <BucketView bucket={bucket!} />
          )}
        </div>
      </div>
    </BucketViewProvider>
  );
}