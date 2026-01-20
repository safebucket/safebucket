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

export function useMFADevices(mfaToken?: string): UseMFADevicesReturn {
  const queryClient = useQueryClient();

  const devicesQuery = useQuery({
    queryKey: ["mfa", "devices"],
    queryFn: () =>
      fetchApi<IMFADevicesResponse>(
        `/mfa/devices`,
        mfaToken ? { headers: { Authorization: `Bearer ${mfaToken}` } } : {},
      ),
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
        `/mfa/devices`,
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
        queryKey: ["mfa", "devices"],
      });
    },
    onError: (error: Error) => errorToast(error),
  });

  const verifyDeviceMutation = useMutation({
    mutationFn: ({ deviceId, code }: { deviceId: string; code: string }) =>
      api.post(
        `/mfa/devices/${deviceId}/verify`,
        { code },
        mfaToken
          ? { headers: { Authorization: `Bearer ${mfaToken}` } }
          : undefined,
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["mfa", "devices"],
      });
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
    }) => api.delete(`/mfa/devices/${deviceId}`, { password }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["mfa", "devices"],
      });
      successToast("MFA device removed");
    },
    onError: (error: Error) => errorToast(error),
  });

  const setDefaultMutation = useMutation({
    mutationFn: (deviceId: string) =>
      api.patch(`/mfa/devices/${deviceId}`, {
        is_default: true,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["mfa", "devices"],
      });
      successToast("MFA device updated");
    },
    onError: (error: Error) => errorToast(error),
  });

  const devices = devicesQuery.data?.devices || [];

  return {
    devices,
    isLoading: devicesQuery.isLoading,
    mfaEnabled: devices.length > 0,
    deviceCount: devices.length,
    maxDevices: MFA_MAX_DEVICES,
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
