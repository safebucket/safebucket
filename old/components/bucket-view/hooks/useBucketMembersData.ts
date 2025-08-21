import { fetchApi } from "@/lib/api";
import useSWR from "swr";

import { IBucketMember } from "@/components/bucket-view/helpers/types";

export interface IBucketMembersData {
  members: IBucketMember[];
  error: string;
  isLoading: boolean;
}

export const useBucketMembersData = (
  bucketId: string | null,
): IBucketMembersData => {
  const { data, error, isLoading } = useSWR(
    bucketId ? `/buckets/${bucketId}/members` : null,
    fetchApi<{ data: IBucketMember[] }>,
  );

  return {
    members: data?.data ?? [],
    error,
    isLoading,
  };
};
