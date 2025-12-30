import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { IUser } from "@/components/auth-view/types/session";
import type { IActivity } from "@/types/activity";
import type {
  AdminStatsResponse,
  CreateUserPayload,
  IAdminBucket,
} from "@/types/admin.ts";
import { api } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

export const usersQueryOptions = () =>
  queryOptions({
    queryKey: ["admin", "users"],
    queryFn: () => api.get<{ data: Array<IUser> }>("/users"),
    select: (data) => data.data,
    staleTime: 5 * 60 * 1000,
  });

export const useCreateUserMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateUserPayload) => api.post<IUser>("/users", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
      successToast("User created successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useDeleteUserMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => api.delete(`/users/${userId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
      successToast("User deleted successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const adminStatsQueryOptions = (days: number = 90) =>
  queryOptions({
    queryKey: ["admin", "stats", days],
    queryFn: () => api.get<AdminStatsResponse>(`/admin/stats?days=${days}`),
  });

export const adminActivityQueryOptions = () =>
  queryOptions({
    queryKey: ["admin", "activity"],
    queryFn: () => api.get<{ data: Array<IActivity> }>("/admin/activity"),
    select: (data) => data.data,
  });

export const adminBucketsQueryOptions = () =>
  queryOptions({
    queryKey: ["admin", "buckets"],
    queryFn: () => api.get<{ data: Array<IAdminBucket> }>("/admin/buckets"),
    select: (data) => data.data,
    staleTime: 60 * 1000,
  });

export const useDeleteAdminBucketMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (bucketId: string) => api.delete(`/buckets/${bucketId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "buckets"] });
      queryClient.invalidateQueries({ queryKey: ["admin", "stats"] });
      successToast("Bucket deleted successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};
