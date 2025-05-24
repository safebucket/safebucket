import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import {
  IActivityData,
  IListBucketActivity,
} from "@/components/activity-view/helpers/types";

export const useActivityData = (): IActivityData => {
  const { data, error, isLoading } = useSWR(
    "/buckets/history",
    fetchApi<IListBucketActivity>,
  );

  return {
    activity: data ? data.data : [],
    error,
    isLoading,
  };
};
