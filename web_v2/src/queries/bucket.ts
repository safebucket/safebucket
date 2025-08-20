import { queryOptions } from "@tanstack/react-query";
import type { ActivityResponse } from "@/types/activity";
import type { IBucket } from "@/components/bucket-view/helpers/types.ts";
import { api } from "@/lib/api";

export const bucketDataQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId],
    queryFn: () => api.get<IBucket>(`/buckets/${bucketId}`),
  });

export const bucketActivityQueryOptions = () =>
  queryOptions({
    queryKey: ["buckets", "activity"],
    queryFn: () => api.get<ActivityResponse>("/buckets/activity"),
    select: (data) => data.data,
  });
