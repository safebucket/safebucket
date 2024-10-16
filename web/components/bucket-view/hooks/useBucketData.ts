import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { IBucket, IBucketData } from "@/components/bucket-view/helpers/types";

export const useBucketData = (id: string): IBucketData => {
  const { data, error, isLoading } = useSWR(
    `/buckets/${id}`,
    fetchApi<IBucket>,
  );

  return {
    bucket: data,
    error,
    isLoading,
  };
};
