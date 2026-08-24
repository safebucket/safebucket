import { useSuspenseQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";

import { SettingsTab } from "@/components/bucket-view/components/SettingsTab";
import { bucketDataQueryOptions } from "@/queries/bucket.ts";

export const Route = createFileRoute(
  "/_authenticated/buckets/$bucketId/settings",
)({
  component: SettingsRoute,
});

function SettingsRoute() {
  const { bucketId } = Route.useParams();
  const bucket = useSuspenseQuery(bucketDataQueryOptions(bucketId)).data;

  return <SettingsTab bucket={bucket} />;
}
