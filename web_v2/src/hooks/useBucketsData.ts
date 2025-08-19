import { api } from "@/lib/api";
import type { Bucket, BucketsResponse, Invites } from "@/types/bucket";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function useBucketsData() {
  const queryClient = useQueryClient();

  const {
    data: bucketsData,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["buckets"],
    queryFn: () => api.get<BucketsResponse>("/buckets"),
    select: (data) => data.data,
  });

  const createBucketMutation = useMutation({
    mutationFn: async ({
      name,
      shareWith,
    }: {
      name: string;
      shareWith: Invites[];
    }) => {
      // Create bucket
      const bucket = await api.post<Bucket>("/buckets", { name });

      // Send invites if any
      if (shareWith.length > 0) {
        await api.post(`/buckets/${bucket.id}/invites`, { invites: shareWith });
      }

      return bucket;
    },
    onSuccess: () => {
      // Invalidate and refetch buckets
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
    },
  });

  const createBucketAndInvites = (name: string, shareWith: Invites[]) => {
    createBucketMutation.mutate({ name, shareWith });
  };

  return {
    buckets: bucketsData || [],
    isLoading,
    error: error?.message || "",
    createBucketAndInvites,
  };
}
