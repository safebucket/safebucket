import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  IUser,
  IMFADevicesResponse,
  IMFADeviceSetupResponse,
} from "@/components/auth-view/types/session";
import { api, fetchApi } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
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

export interface MFAResetRequestResponse {
  challenge_id: string;
}

export const useRequestMFAResetMutation = (userId: string) => {
  return useMutation({
    mutationFn: (password: string) =>
      api.post<MFAResetRequestResponse>(`/users/${userId}/mfa/reset`, {
        password,
      }),
    onError: (error: Error) => errorToast(error),
  });
};

export const useVerifyMFAResetMutation = (
  userId: string,
  challengeId: string,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (code: string) =>
      api.post(`/users/${userId}/mfa/reset/${challengeId}`, { code }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      queryClient.invalidateQueries({ queryKey: ["users", userId, "mfa"] });
      successToast("MFA has been reset successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useMFADevices = (userId: string) => {
  return useQuery({
    queryKey: ["users", userId, "mfa", "devices"],
    queryFn: () =>
      fetchApi<IMFADevicesResponse>(`/users/${userId}/mfa/devices`),
    enabled: !!userId,
    staleTime: 5 * 60 * 1000,
  });
};

export const useAddMFADeviceMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) =>
      api.post<IMFADeviceSetupResponse>(`/users/${userId}/mfa/devices`, {
        name,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useVerifyMFADeviceMutation = (
  userId: string,
  deviceId: string,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (code: string) =>
      api.post(`/users/${userId}/mfa/devices/${deviceId}/verify`, { code }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      successToast("MFA device verified successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useRemoveMFADeviceMutation = (userId: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      deviceId,
      password,
    }: {
      deviceId: string;
      password: string;
    }) => api.delete(`/users/${userId}/mfa/devices/${deviceId}`, { password }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      successToast("MFA device removed");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useUpdateMFADeviceMutation = (
  userId: string,
  deviceId: string,
) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { name?: string; is_default?: boolean }) =>
      api.patch(`/users/${userId}/mfa/devices/${deviceId}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      successToast("MFA device updated");
    },
    onError: (error: Error) => errorToast(error),
  });
};
