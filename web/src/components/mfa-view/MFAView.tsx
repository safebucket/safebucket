import { Card, CardContent } from "@/components/ui/card";
import { MFAViewProvider } from "@/components/mfa-view/context/MFAViewProvider";
import { MFAHeader } from "@/components/mfa-view/components/MFAHeader";
import { MFADeviceList } from "@/components/mfa-view/components/MFADeviceList";
import { MFAEmptyState } from "@/components/mfa-view/components/MFAEmptyState";
import { MFAActions } from "@/components/mfa-view/components/MFAActions";
import { MFASetupDialog } from "@/components/mfa-view/components/MFASetupDialog";
import { MFADeleteDialog } from "@/components/mfa-view/components/MFADeleteDialog";
import { MFAResetDialog } from "@/components/mfa-view/components/MFAResetDialog";
import { useMFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";

interface MFAViewProps {
  userId: string;
  className?: string;
}

function MFAViewContent() {
  const { devices, isLoading } = useMFAViewContext();

  if (isLoading) {
    return null;
  }

  if (devices.length === 0) {
    return <MFAEmptyState />;
  }

  return (
    <>
      <MFADeviceList />
      <MFAActions />
    </>
  );
}

export function MFAView({ userId, className }: MFAViewProps) {
  return (
    <MFAViewProvider userId={userId}>
      <Card className={className}>
        <MFAHeader />
        <CardContent className="space-y-4">
          <MFAViewContent />
        </CardContent>
      </Card>
      <MFASetupDialog />
      <MFADeleteDialog />
      <MFAResetDialog />
    </MFAViewProvider>
  );
}
