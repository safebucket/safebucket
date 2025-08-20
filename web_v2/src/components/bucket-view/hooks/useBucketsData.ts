import { useState } from "react";

import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import {
  api_createBucket,
  api_createInvites,
} from "@/components/bucket-view/helpers/api";
import type {
  IBucket,
  IBucketsData,
  IMembers,
  IListBuckets,
} from "@/components/bucket-view/helpers/types";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data, error, isLoading, mutate } = useSWR(
    "/buckets",
    fetchApi<IListBuckets>,
  );

  const createBucketAndInvites = async (name: string, invites: IMembers[]) => {
    api_createBucket(name)
      .then((bucket: IBucket) =>
        api_createInvites(bucket.id, invites).then(() => {
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
    createBucketAndInvites,
  };
};
