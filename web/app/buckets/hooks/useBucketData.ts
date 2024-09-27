import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { Bucket, IBucketData } from "@/app/buckets/helpers/types";

export const useBucketData = (id: string): IBucketData => {
  const { data, error, isLoading } = useSWR(
    `/buckets/${id}`,
    fetchApi<Bucket>,
  );

  return {
    bucket: data,
    error,
    isLoading,
  };
};
