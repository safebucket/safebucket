import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import {
  IActivityData,
  IListBucketActivity,
} from "@/components/common/types/activity";

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
