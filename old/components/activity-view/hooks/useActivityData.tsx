import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import {
  IActivityData,
  IListBucketActivity,
} from "@/components/common/types/activity";

export const useActivityData = (): IActivityData => {
  const { data, error, isLoading } = useSWR(
    "/buckets/activity",
    fetchApi<IListBucketActivity>,
  );

  return {
    activity: data ? data.data : [],
    error,
    isLoading,
  };
};
