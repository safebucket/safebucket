import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import type { IAdminBucket } from "@/queries/admin";
import {
  adminBucketsQueryOptions,
  useDeleteAdminBucketMutation,
} from "@/queries/admin";

export const useAdminBucketsData = () => {
  const [bucketToDelete, setBucketToDelete] = useState<IAdminBucket | null>(
    null,
  );

  const { data: buckets, isLoading } = useQuery(adminBucketsQueryOptions());
  const deleteBucketMutation = useDeleteAdminBucketMutation();

  return {
    buckets: buckets ?? [],
    isLoading,
    deleteBucketMutation,
    bucketToDelete,
    setBucketToDelete,
  };
};
