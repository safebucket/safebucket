import { useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { api_createBucket } from "@/components/bucket-view/helpers/api";
import {
  IBucketsData,
  IListBuckets,
  IShareWith,
} from "@/components/bucket-view/helpers/types";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data, error, isLoading, mutate } = useSWR(
    "/buckets",
    fetchApi<IListBuckets>,
  );

  const createBucket = async (name: string, shareWith: IShareWith[]) => {
    api_createBucket(name, shareWith)
      .then(() => {
        mutate();
        setIsDialogOpen(false);
        successToast("The bucket has been created");
      })
      .catch(errorToast);
  };

  return {
    buckets: data ? data.data : [],
    error,
    isLoading,
    isDialogOpen,
    setIsDialogOpen,
    createBucket,
  };
};
