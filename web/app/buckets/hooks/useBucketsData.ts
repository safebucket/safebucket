import useSWR from "swr";

import { IBucketsData } from "@/app/buckets/helpers/types";
import { fetcher } from "@/app/helpers/utils";

export const useBucketsData = (): IBucketsData => {
  const { data, error, isLoading } = useSWR("/buckets", fetcher);

  return {
    buckets: data?.data,
    error,
    isLoading,
  };
};
