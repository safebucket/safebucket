import { useState } from "react";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type {
  IBucketsData,
  IMembers,
} from "@/components/bucket-view/helpers/types";

import type { IBucket } from "@/types/bucket.ts";
import { bucketsQueryOptions } from "@/queries/bucket.ts";
import { api } from "@/lib/api.ts";

export const useBucketsData = (): IBucketsData => {
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { data: buckets, isLoading } = useQuery(bucketsQueryOptions());

  const queryClient = useQueryClient();

  const createBucketMutation = useMutation({
    mutationFn: async ({
      name,
      members,
    }: {
      name: string;
      members: Array<IMembers>;
    }) => {
      const bucket = await api.post<IBucket>("/buckets", { name });

      if (members.length > 0) {
        await api.put(`/buckets/${bucket.id}/members`, {
          members,
        });
      }

      return bucket;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      setIsDialogOpen(false);
      toast.success("The bucket has been created");
    },
  });

  return {
    buckets: buckets ? buckets : [],
    isLoading,
    isDialogOpen,
    setIsDialogOpen,
    createBucketMutation,
  };
};
