import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { IActivity, IListBucketActivity } from "@/types/activity";
import type {
  IBucketMember,
  INotificationPreferences,
} from "@/components/bucket-view/helpers/types.ts";
import type { IBucket } from "@/types/bucket.ts";
import type { IShare, IShareCreateBody } from "@/types/share.ts";
import { api } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import i18n from "@/lib/i18n";

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

export const useUpdateNotificationPreferencesMutation = (bucketId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: INotificationPreferences) =>
      api.patch(`/buckets/${bucketId}/members/notifications`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "members"],
      });
      successToast(i18n.t("bucket.notifications.updated"));
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const bucketSharesQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "shares"],
    queryFn: () =>
      api.get<{ data: Array<IShare> }>(`/buckets/${bucketId}/shares`),
    select: (response) => response.data,
  });

export const useCreateShareMutation = (bucketId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: IShareCreateBody) =>
      api.post<IShare>(`/buckets/${bucketId}/shares`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "shares"],
      });
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useDeleteShareMutation = (bucketId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (shareId: string) =>
      api.delete(`/buckets/${bucketId}/shares/${shareId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["buckets", bucketId, "shares"],
      });
      successToast(i18n.t("bucket.settings.shares.deleted"));
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const bucketTrashedFilesQueryOptions = (bucketId: string) =>
  queryOptions({
    queryKey: ["buckets", bucketId, "trash"],
    queryFn: async () => {
      const response = await api.get<IBucket>(
        `/buckets/${bucketId}?status=deleted`,
      );
      return {
        files: response.files,
        folders: response.folders,
      };
    },
    enabled: !!bucketId,
  });
