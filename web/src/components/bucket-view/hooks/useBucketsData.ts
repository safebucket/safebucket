import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import type { IBucketsData } from "@/components/bucket-view/helpers/types";

import type { IBucket } from "@/types/bucket.ts";
import { bucketsQueryOptions } from "@/queries/bucket.ts";
import { api } from "@/lib/api.ts";
import i18n from "@/lib/i18n";

export const useBucketsData = (): IBucketsData => {
  const { data: buckets, isLoading } = useQuery(bucketsQueryOptions());

  const queryClient = useQueryClient();

  const createBucketMutation = useMutation({
    mutationFn: ({ name }: { name: string }) =>
      api.post<IBucket>("/buckets", { name }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["buckets"] });
      toast.success(i18n.t("bucket.new_bucket_dialog.created"));
    },
  });

  return {
    buckets: buckets ? buckets : [],
    isLoading,
    createBucketMutation,
  };
};
