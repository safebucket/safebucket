import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/_authenticated/buckets/$bucketId/")({
  beforeLoad: ({ params: { bucketId } }) => {
    throw redirect({
      to: "/buckets/$bucketId/files/{-$folderId}",
      params: { bucketId },
    });
  },
});
