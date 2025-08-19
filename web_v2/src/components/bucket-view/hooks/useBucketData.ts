import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import type { IBucket, IBucketData } from "@/components/bucket-view/helpers/types";

export const useBucketData = (id: string): IBucketData => {
  const { data, error, isLoading } = useSWR(
    `/buckets/${id}`,
    fetchApi<IBucket>,
  );

  return {
    bucket: data ?? undefined,
    error,
    isLoading,
  };
};
