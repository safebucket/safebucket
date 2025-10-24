import { queryOptions } from "@tanstack/react-query";
import type { IActivity, IListBucketActivity } from "@/types/activity";
import type { IBucketMember } from "@/components/bucket-view/helpers/types.ts";
import type { IBucket } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import { api } from "@/lib/api";

export const bucketsQueryOptions = () =>
  queryOptions({
    queryKey: ["buckets"],
    queryFn: () => api.get<{ data: Array<IBucket> }>("/buckets"),
    select: (data) => data.data,
  });

export const bucketsActivityQueryOptions = () =>
  queryOptions({
    queryKey: ["buckets", "activity"],
    queryFn: () => api.get<IListBucketActivity>("/buckets/activity"),
    select: (data) => data.data,
  });

export const bucketDataQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId],
    queryFn: () => api.get<IBucket>(`/buckets/${bucketId}`),
  });

export const bucketActivityQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "activity"],
    queryFn: () =>
      api.get<{ data: Array<IActivity> }>(`/buckets/${bucketId}/activity`),
    select: (response) => response.data,
  });

export const bucketMembersQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "members"],
    queryFn: () =>
      api.get<{ data: Array<IBucketMember> }>(`/buckets/${bucketId}/members`),
    select: (response) => response.data,
  });

export const bucketTrashedFilesQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "trash"],
    queryFn: async () => {
      const response = await api.get<{ data: Array<IFile> }>(
        `/buckets/${bucketId}/trash`,
      );
      return response.data;
    },
    enabled: !!bucketId,
  });
