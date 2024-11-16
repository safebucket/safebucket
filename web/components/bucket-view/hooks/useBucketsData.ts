import { useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { api_createBucket } from "@/components/bucket-view/helpers/api";
import {
  IBucketsData,
  IListBuckets,
} from "@/components/bucket-view/helpers/types";
import { useToast } from "@/components/common/hooks/use-toast";

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const { toast } = useToast();

  const { data, error, isLoading, mutate } = useSWR(
    "/buckets",
    fetchApi<IListBuckets>,
  );

  const createBucket = async (name: string) => {
    api_createBucket(name).then(() => {
      mutate();
      setIsDialogOpen(false);
      toast({
        variant: "success",
        title: "Success",
        description: "The bucket has been created",
      });
    });
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
