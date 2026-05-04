import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  ISessionListResponse,
  IUser,
} from "@/components/auth-view/types/session";
import { api, fetchApi } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import i18n from "@/lib/i18n";
import { useSession } from "@/hooks/useAuth";

interface UpdateUserPayload {
  first_name?: string;
  last_name?: string;
  old_password?: string;
  new_password?: string;
}

export interface UserStats {
  total_files: number;
  total_buckets: number;
}

export const useCurrentUser = () => {
  const session = useSession();

  return useQuery({
    queryKey: ["users", session?.userId],
    queryFn: () => fetchApi<IUser>(`/users/${session?.userId}`),
    enabled: !!session?.userId,
    staleTime: 5 * 60 * 1000,
  });
};

export const useUpdateUserMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateUserPayload) =>
      api.patch(`/users/${userId}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      successToast("Profile updated successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useUserStatsQuery = (userId: string) => {
  return useQuery({
    queryKey: ["users", userId, "stats"],
    queryFn: () => fetchApi<UserStats>(`/users/${userId}/stats`),
    enabled: !!userId,
    staleTime: 15 * 60 * 1000,
  });
};

export const useSessionsQuery = (userId: string) => {
  return useQuery({
    queryKey: ["users", userId, "sessions"],
    queryFn: () => fetchApi<ISessionListResponse>(`/users/${userId}/sessions`),
    enabled: !!userId,
    staleTime: 30 * 1000,
  });
};

export const useRevokeSessionMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (sessionId: string) =>
      api.delete(`/users/${userId}/sessions/${sessionId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "sessions"],
      });
      successToast(i18n.t("settings.sessions.revoked"));
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useRevokeAllSessionsMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.delete(`/users/${userId}/sessions`),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "sessions"],
      });
      successToast(i18n.t("settings.sessions.all_revoked"));
    },
    onError: (error: Error) => errorToast(error),
  });
};
