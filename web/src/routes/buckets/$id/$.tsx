import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { bucketDataQueryOptions } from "@/queries/bucket.ts";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";
import { FileType } from "@/types/file.ts";

export const Route = createFileRoute("/buckets/$id/$")({
  loader: ({ context: { queryClient }, params: { id } }) =>
    queryClient.ensureQueryData(bucketDataQueryOptions(id)),
  component: BucketComponent,
});

function BucketComponent() {
  const { id, _splat } = Route.useParams();
  const bucketQuery = useSuspenseQuery(bucketDataQueryOptions(id));
  const bucket = bucketQuery.data;

  // If there's a splat path, validate that it exists as a folder
  if (_splat) {
    const segments = _splat.split("/");
    const folderName = segments[segments.length - 1];
    const parentPath =
      segments.length === 1 ? "/" : `/${segments.slice(0, -1).join("/")}`;

    const folderExists = bucket.files.some(
      (f) =>
        f.name === folderName &&
        f.path === parentPath &&
        f.type === FileType.folder,
    );

    if (!folderExists) {
      window.location.href = `/buckets/${id}`;
    }
  }

  const joinedPath = _splat ? `/${_splat}` : "/";

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
