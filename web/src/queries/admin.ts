import {
  infiniteQueryOptions,
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { toast } from "sonner";
import type { IUser } from "@/components/auth-view/types/session";
import type { ActivityMessage, IActivityPage } from "@/types/activity";
import type {
  AdminStatsResponse,
  CreateUserPayload,
  IAdminBucket,
} from "@/types/admin.ts";
import type { IAdminSettingsResponse } from "@/types/app_settings";
import { api } from "@/lib/api";

const ACTIVITY_PAGE_SIZE = 50;

export interface AdminActivityFilters {
  action?: Array<ActivityMessage>;
  type?: Array<string>;
  from?: string;
  to?: string;
}

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
      toast.success("User created successfully");
    },
  });
};

export const useDeleteUserMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (userId: string) => api.delete(`/users/${userId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
      toast.success("User deleted successfully");
    },
  });
};

export const adminStatsQueryOptions = (days: number = 90) =>
  queryOptions({
    queryKey: ["admin", "stats", days],
    queryFn: () => api.get<AdminStatsResponse>(`/admin/stats?days=${days}`),
  });

export const adminSettingsQueryOptions = () =>
  queryOptions({
    queryKey: ["admin", "settings"],
    queryFn: () => api.get<IAdminSettingsResponse>("/admin/settings"),
    staleTime: 5 * 60 * 1000,
  });

export const adminActivityInfiniteQueryOptions = (
  filters: AdminActivityFilters = {},
) =>
  infiniteQueryOptions({
    queryKey: ["admin", "activity", filters],
    queryFn: ({ pageParam }) =>
      api.get<IActivityPage>("/admin/activity", {
        params: {
          action: filters.action?.length ? filters.action.join(",") : undefined,
          type: filters.type?.length ? filters.type.join(",") : undefined,
          from: filters.from,
          to: filters.to,
          limit: ACTIVITY_PAGE_SIZE,
          cursor: pageParam ?? undefined,
        },
      }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
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
      toast.success("Bucket deleted successfully");
    },
  });
};
