import { api } from "@/lib/api";
import type { ActivityResponse } from "@/types/activity";
import { queryOptions } from "@tanstack/react-query";

export const bucketActivityQueryOptions = () =>
  queryOptions({
    queryKey: ["buckets", "activity"],
    queryFn: () => api.get<ActivityResponse>("/buckets/activity"),
    select: (data) => data.data,
  });
