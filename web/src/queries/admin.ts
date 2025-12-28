import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { IUser } from "@/components/auth-view/types/session";
import type { IActivity } from "@/types/activity";
import { api } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

export interface CreateUserPayload {
  first_name: string;
  last_name: string;
  email: string;
  password: string;
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

export interface RoleCount {
  role: string;
  count: number;
}

export interface ProviderCount {
  provider: string;
  count: number;
}

export interface TimeSeriesPoint {
  date: string;
  count: number;
}

export interface AdminStatsResponse {
  total_users: number;
  total_buckets: number;
  total_files: number;
  total_folders: number;
  total_storage: number;
  role_distribution: Array<RoleCount>;
  provider_distribution: Array<ProviderCount>;
  shared_files: Array<TimeSeriesPoint>;
}

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
