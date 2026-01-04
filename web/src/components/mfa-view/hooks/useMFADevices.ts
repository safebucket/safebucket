import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type {
  IMFADevice,
  IMFADeviceSetupResponse,
  IMFADevicesResponse,
} from "@/components/mfa-view/helpers/types";
import { api, fetchApi } from "@/lib/api";
import { errorToast, successToast } from "@/components/ui/hooks/use-toast";
import { MFA_MAX_DEVICES } from "@/components/mfa-view/helpers/constants";

export interface UseMFADevicesReturn {
  devices: Array<IMFADevice>;
  isLoading: boolean;
  mfaEnabled: boolean;
  deviceCount: number;
  maxDevices: number;
  addDevice: (
    name: string,
    password?: string,
    mfaToken?: string,
  ) => Promise<IMFADeviceSetupResponse>;
  verifyDevice: (deviceId: string, code: string) => Promise<unknown>;
  removeDevice: (deviceId: string, password: string) => Promise<void>;
  setDefault: (deviceId: string) => Promise<void>;
  isAddingDevice: boolean;
  isVerifyingDevice: boolean;
  isRemovingDevice: boolean;
}

export function useMFADevices(
  userId: string,
  mfaToken?: string,
): UseMFADevicesReturn {
  const queryClient = useQueryClient();

  const devicesQuery = useQuery({
    queryKey: ["users", userId, "mfa", "devices"],
    queryFn: () =>
      fetchApi<IMFADevicesResponse>(
        `/users/${userId}/mfa/devices`,
        mfaToken ? { headers: { Authorization: `Bearer ${mfaToken}` } } : {},
      ),
    enabled: !!userId,
    staleTime: 5 * 60 * 1000,
  });

  const addDeviceMutation = useMutation({
    mutationFn: ({
      name,
      password,
      mfaToken,
    }: {
      name: string;
      password?: string;
      mfaToken?: string;
    }) =>
      api.post<IMFADeviceSetupResponse>(
        `/users/${userId}/mfa/devices`,
        {
          name,
          password,
        },
        mfaToken
          ? { headers: { Authorization: `Bearer ${mfaToken}` } }
          : undefined,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
    },
    onError: (error: Error) => errorToast(error),
  });

  const verifyDeviceMutation = useMutation({
    mutationFn: ({ deviceId, code }: { deviceId: string; code: string }) =>
      api.post(
        `/users/${userId}/mfa/devices/${deviceId}/verify`,
        { code },
        mfaToken
          ? { headers: { Authorization: `Bearer ${mfaToken}` } }
          : undefined,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      queryClient.invalidateQueries({ queryKey: ["users", userId] });
      successToast("MFA device verified successfully");
    },
    onError: (error: Error) => errorToast(error),
  });

  const removeDeviceMutation = useMutation({
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

  const setDefaultMutation = useMutation({
    mutationFn: (deviceId: string) =>
      api.patch(`/users/${userId}/mfa/devices/${deviceId}`, {
        is_default: true,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["users", userId, "mfa", "devices"],
      });
      successToast("MFA device updated");
    },
    onError: (error: Error) => errorToast(error),
  });

  return {
    devices: devicesQuery.data?.devices || [],
    isLoading: devicesQuery.isLoading,
    mfaEnabled: devicesQuery.data?.mfa_enabled || false,
    deviceCount: devicesQuery.data?.device_count || 0,
    maxDevices: devicesQuery.data?.max_devices || MFA_MAX_DEVICES,
    addDevice: async (name: string, password?: string, mfaToken?: string) => {
      return addDeviceMutation.mutateAsync({ name, password, mfaToken });
    },
    verifyDevice: (deviceId: string, code: string) =>
      verifyDeviceMutation.mutateAsync({ deviceId, code }),
    removeDevice: async (deviceId: string, password: string) => {
      await removeDeviceMutation.mutateAsync({ deviceId, password });
    },
    setDefault: async (deviceId: string) => {
      await setDefaultMutation.mutateAsync(deviceId);
    },
    isAddingDevice: addDeviceMutation.isPending,
    isVerifyingDevice: verifyDeviceMutation.isPending,
    isRemovingDevice: removeDeviceMutation.isPending,
  };
}
