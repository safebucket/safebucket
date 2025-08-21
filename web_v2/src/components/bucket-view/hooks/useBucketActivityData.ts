import useSWR from "swr";

import type { IActivityData, IListBucketActivity } from "@/types/activity.ts";
import { fetchApi } from "@/lib/api";

export const useBucketActivityData = (id: string): IActivityData => {
  const { data, error, isLoading } = useSWR(
    `/buckets/${id}/activity`,
    fetchApi<IListBucketActivity>,
  );

  return {
    activity: data ? data.data : [],
    error,
    isLoading,
  };
};
