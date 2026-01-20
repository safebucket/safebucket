import { useTranslation } from "react-i18next";
import { Plus } from "lucide-react";

import { Button } from "@/components/ui/button";

interface MFAActionsProps {
  deviceCount: number;
  maxDevices: number;
  onAddDevice: () => void;
}

export function MFAActions({
  deviceCount,
  maxDevices,
  onAddDevice,
}: MFAActionsProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center pt-4 border-t">
      <Button
        variant="outline"
        onClick={onAddDevice}
        disabled={deviceCount >= maxDevices}
      >
        <Plus className="mr-2 h-4 w-4" />
        {t("auth.mfa.add_device")}
      </Button>
    </div>
  );
}
