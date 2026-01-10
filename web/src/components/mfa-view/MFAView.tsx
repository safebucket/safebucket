import { useState } from "react";

import { Card, CardContent } from "@/components/ui/card";
import { MFAHeader } from "@/components/mfa-view/components/MFAHeader";
import { MFADeviceList } from "@/components/mfa-view/components/MFADeviceList";
import { MFAEmptyState } from "@/components/mfa-view/components/MFAEmptyState";
import { MFAActions } from "@/components/mfa-view/components/MFAActions";
import { MFASetupDialog } from "@/components/mfa-view/components/MFASetupDialog";
import { MFADeleteDialog } from "@/components/mfa-view/components/MFADeleteDialog";
import { MFAResetDialog } from "@/components/mfa-view/components/MFAResetDialog";
import { useMFADevices } from "@/components/mfa-view/hooks/useMFADevices";

interface MFAViewProps {
  userId: string;
  className?: string;
}

export function MFAView({ userId, className }: MFAViewProps) {
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

  const closeAllDialogs = () => {
    setSetupDialogOpen(false);
    setDeleteDeviceId(null);
    setResetDialogOpen(false);
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
          onReset={() => setResetDialogOpen(true)}
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
        userId={userId}
        open={setupDialogOpen}
        onClose={closeAllDialogs}
      />
      <MFADeleteDialog
        userId={userId}
        deviceId={deleteDeviceId}
        onClose={closeAllDialogs}
      />
      <MFAResetDialog
        userId={userId}
        open={resetDialogOpen}
        onClose={closeAllDialogs}
      />
    </>
  );
}
