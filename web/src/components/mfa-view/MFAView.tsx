import { useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent } from "@/components/ui/card";
import { MFAHeader } from "@/components/mfa-view/components/MFAHeader";
import { MFADeviceList } from "@/components/mfa-view/components/MFADeviceList";
import { MFAEmptyState } from "@/components/mfa-view/components/MFAEmptyState";
import { MFAActions } from "@/components/mfa-view/components/MFAActions";
import { MFASetupDialog } from "@/components/mfa-view/components/MFASetupDialog";
import { MFADeleteDialog } from "@/components/mfa-view/components/MFADeleteDialog";
import {
  mfaDevicesQueryOptions,
  useSetDefaultMFADeviceMutation,
} from "@/queries/mfa";

interface MFAViewProps {
  className?: string;
  providerType: string;
}

export function MFAView({ className, providerType }: MFAViewProps) {
  const { data, isLoading } = useQuery(mfaDevicesQueryOptions());
  const setDefaultMutation = useSetDefaultMFADeviceMutation();

  const devices = data?.devices ?? [];
  const maxDevices = data?.max_devices ?? 0;
  const deviceCount = devices.length;
  const mfaEnabled = deviceCount > 0;

  const [setupDialogOpen, setSetupDialogOpen] = useState(false);
  const [deleteDeviceId, setDeleteDeviceId] = useState<string | null>(null);

  const closeAllDialogs = () => {
    setSetupDialogOpen(false);
    setDeleteDeviceId(null);
  };

  const renderContent = () => {
    if (isLoading) {
      return null;
    }

    if (devices.length === 0) {
      return <MFAEmptyState onSetup={() => setSetupDialogOpen(true)} />;
    }

    return (
      <>
        <MFADeviceList
          devices={devices}
          onSetDefault={(id) => setDefaultMutation.mutate(id)}
          onDelete={setDeleteDeviceId}
        />
        <MFAActions
          deviceCount={deviceCount}
          maxDevices={maxDevices}
          onAddDevice={() => setSetupDialogOpen(true)}
        />
      </>
    );
  };

  return (
    <>
      <Card className={className}>
        <MFAHeader
          mfaEnabled={mfaEnabled}
          deviceCount={deviceCount}
          maxDevices={maxDevices}
        />
        <CardContent className="space-y-4">{renderContent()}</CardContent>
      </Card>
      <MFASetupDialog
        open={setupDialogOpen}
        onClose={closeAllDialogs}
        providerType={providerType}
        hasExistingDevices={deviceCount > 0}
      />
      <MFADeleteDialog
        deviceId={deleteDeviceId}
        onClose={closeAllDialogs}
        providerType={providerType}
      />
    </>
  );
}
