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
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";

const MFA_DEVICES_KEY = ["mfa", "devices"] as const;

const authHeaders = (mfaToken?: string) =>
  mfaToken ? { headers: { Authorization: `Bearer ${mfaToken}` } } : undefined;

export const mfaDevicesQueryOptions = (mfaToken?: string) =>
  queryOptions({
    queryKey: MFA_DEVICES_KEY,
    queryFn: () =>
      fetchApi<IMFADevicesResponse>(
        `/mfa/devices`,
        authHeaders(mfaToken) ?? {},
      ),
    staleTime: 5 * 60 * 1000,
  });

export const useAddMFADeviceMutation = (mfaToken?: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ name, password }: { name: string; password?: string }) =>
      api.post<IMFADeviceSetupResponse>(
        `/mfa/devices`,
        { name, password },
        authHeaders(mfaToken),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useVerifyMFADeviceMutation = (mfaToken?: string) => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ deviceId, code }: { deviceId: string; code: string }) =>
      api.post<{ access_token?: string; refresh_token?: string }>(
        `/mfa/devices/${deviceId}/verify`,
        { code },
        authHeaders(mfaToken),
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
      successToast("MFA device verified successfully");
    },
    onError: (error: Error) => errorToast(error),
  });
};

export const useRemoveMFADeviceMutation = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      deviceId,
      password,
    }: {
      deviceId: string;
      password: string;
    }) => api.delete(`/mfa/devices/${deviceId}`, { password }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MFA_DEVICES_KEY });
      successToast("MFA device removed");
    },
    onError: (error: Error) => errorToast(error),
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
    onError: (error: Error) => errorToast(error),
  });
};
