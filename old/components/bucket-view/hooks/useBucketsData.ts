import { useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import {
  api_addMembers,
  api_createBucket,
} from "@/components/bucket-view/helpers/api";
import {
  IBucket,
  IBucketsData,
  IListBuckets,
  IMembers,
} from "@/components/bucket-view/helpers/types";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data, error, isLoading, mutate } = useSWR(
    "/buckets",
    fetchApi<IListBuckets>,
  );

  const createBucketAndMembers = async (name: string, members: IMembers[]) => {
    api_createBucket(name)
      .then((bucket: IBucket) =>
        api_addMembers(bucket.id, members).then(() => {
          mutate();
          setIsDialogOpen(false);
          successToast("The bucket has been created");
        }),
      )
      .catch(errorToast);
  };

  return {
    buckets: data ? data.data : [],
    error,
    isLoading,
    isDialogOpen,
    setIsDialogOpen,
    createBucketAndMembers,
  };
};
