import useSWR from "swr";

import { IBucketData } from "@/app/buckets/helpers/types";
import { fetcher } from "@/app/helpers/utils";

export const useBucketData = (id: string): IBucketData => {
  const { data, error, isLoading } = useSWR(`/buckets/${id}`, fetcher);

  return {
    bucket: data,
    error,
    isLoading,
  };
};
