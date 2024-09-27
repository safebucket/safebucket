import { useState } from "react";

import { useToast } from "@/components/hooks/use-toast";
import { api, fetchApi } from "@/lib/api";
import { SubmitHandler, useForm } from "react-hook-form";
import useSWR from "swr";

import { Bucket, IBucketForm, IBucketsData } from "@/app/buckets/helpers/types";

export type IListBuckets = {
  data: Bucket[];
};

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const { toast } = useToast();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<IBucketForm>();

  const { data, error, isLoading, mutate } = useSWR(
    "/buckets",
    fetchApi<IListBuckets>,
  );

  const createBucket: SubmitHandler<IBucketForm> = async (body) => {
    await api.post("/buckets", body).then(() => {
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
    register,
    handleSubmit,
  };
};
