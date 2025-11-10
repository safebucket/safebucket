import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IUser } from "@/components/auth-view/types/session";
import { api, fetchApi } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";

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
  const { session } = useSessionContext();

  return useQuery({
    queryKey: ["users", session?.userId],
    queryFn: () => fetchApi<IUser>(`/users/${session?.userId}`),
    enabled: !!session?.userId,
    staleTime: 5 * 60 * 1000, // Consider data fresh for 5 minutes
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
    staleTime: 15 * 60 * 1000, // Consider data fresh for 15 minutes
  });
};
