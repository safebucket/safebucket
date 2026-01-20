import { useState } from "react";

import { Card, CardContent } from "@/components/ui/card";
import { MFAHeader } from "@/components/mfa-view/components/MFAHeader";
import { MFADeviceList } from "@/components/mfa-view/components/MFADeviceList";
import { MFAEmptyState } from "@/components/mfa-view/components/MFAEmptyState";
import { MFAActions } from "@/components/mfa-view/components/MFAActions";
import { MFASetupDialog } from "@/components/mfa-view/components/MFASetupDialog";
import { MFADeleteDialog } from "@/components/mfa-view/components/MFADeleteDialog";
import { useMFADevices } from "@/components/mfa-view/hooks/useMFADevices";

interface MFAViewProps {
  className?: string;
}

export function MFAView({ className }: MFAViewProps) {
  const {
    devices,
    isLoading,
    mfaEnabled,
    deviceCount,
    maxDevices,
    setDefault,
  } = useMFADevices();

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
          onSetDefault={setDefault}
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
      <MFASetupDialog open={setupDialogOpen} onClose={closeAllDialogs} />
      <MFADeleteDialog deviceId={deleteDeviceId} onClose={closeAllDialogs} />
    </>
  );
}
