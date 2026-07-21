import {
  queryOptions,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type {
  IMFADeviceSetupResponse,
  IMFADevicesResponse,
} from "@/components/auth-view/types/session";
import { api, fetchApi } from "@/lib/api";
import { successToast } from "@/components/ui/hooks/use-toast";

const MFA_DEVICES_KEY = ["mfa", "devices"] as const;

export const mfaDevicesQueryOptions = () =>
  queryOptions({
    queryKey: MFA_DEVICES_KEY,
    queryFn: () => fetchApi<IMFADevicesResponse>(`/mfa/devices`),
    staleTime: 5 * 60 * 1000,
  });

export const useAddMFADeviceMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      name,
      password,
      code,
    }: {
      name: string;
      password?: string;
      code?: string;
    }) =>
      api.post<IMFADeviceSetupResponse>(
        `/mfa/devices`,
        {
          name,
          password,
          code,
        },
        { retryOnRateLimit: false },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
    },
  });
};

export const useVerifyMFADeviceMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ deviceId, code }: { deviceId: string; code: string }) =>
      api.post<{ access_token?: string; refresh_token?: string }>(
        `/mfa/devices/${deviceId}/verify`,
        { code },
        { retryOnRateLimit: false },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
      successToast("MFA device verified successfully");
    },
  });
};

export const useRemoveMFADeviceMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      deviceId,
      password,
      code,
    }: {
      deviceId: string;
      password?: string;
      code?: string;
    }) =>
      api.delete(
        `/mfa/devices/${deviceId}`,
        { password, code },
        { retryOnRateLimit: false },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
      successToast("MFA device removed");
    },
  });
};

export const useSetDefaultMFADeviceMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (deviceId: string) =>
      api.patch(`/mfa/devices/${deviceId}`, { is_default: true }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
      successToast("MFA device updated");
    },
  });
};
