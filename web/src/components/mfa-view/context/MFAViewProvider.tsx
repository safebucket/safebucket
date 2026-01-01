import React, { useCallback, useState } from "react";

import { MFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";
import { useMFADevices } from "@/components/mfa-view/hooks/useMFADevices";

interface MFAViewProviderProps {
  userId: string;
  children: React.ReactNode;
}

export function MFAViewProvider({ userId, children }: MFAViewProviderProps) {
  const {
    devices,
    isLoading,
    mfaEnabled,
    deviceCount,
    maxDevices,
    setDefault,
  } = useMFADevices(userId);

  const [setupDialogOpen, setSetupDialogOpen] = useState(false);
  const [deleteDeviceId, setDeleteDeviceId] = useState<string | null>(null);
  const [resetDialogOpen, setResetDialogOpen] = useState(false);

  const openSetupDialog = useCallback(() => {
    setSetupDialogOpen(true);
  }, []);

  const openDeleteDialog = useCallback((deviceId: string) => {
    setDeleteDeviceId(deviceId);
  }, []);

  const openResetDialog = useCallback(() => {
    setResetDialogOpen(true);
  }, []);

  const closeAllDialogs = useCallback(() => {
    setSetupDialogOpen(false);
    setDeleteDeviceId(null);
    setResetDialogOpen(false);
  }, []);

  const setDeviceDefault = useCallback(
    async (deviceId: string) => {
      await setDefault(deviceId);
    },
    [setDefault],
  );

  return (
    <MFAViewContext.Provider
      value={{
        userId,
        devices,
        isLoading,
        mfaEnabled,
        deviceCount,
        maxDevices,
        setupDialogOpen,
        deleteDeviceId,
        resetDialogOpen,
        openSetupDialog,
        openDeleteDialog,
        openResetDialog,
        closeAllDialogs,
        setDeviceDefault,
      }}
    >
      {children}
    </MFAViewContext.Provider>
  );
}
