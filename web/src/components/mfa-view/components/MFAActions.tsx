import { useTranslation } from "react-i18next";
import { Plus, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";

interface MFAActionsProps {
  deviceCount: number;
  maxDevices: number;
  onAddDevice: () => void;
  onReset: () => void;
}

export function MFAActions({
  deviceCount,
  maxDevices,
  onAddDevice,
  onReset,
}: MFAActionsProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between pt-4 border-t">
      <Button
        variant="outline"
        onClick={onAddDevice}
        disabled={deviceCount >= maxDevices}
      >
        <Plus className="mr-2 h-4 w-4" />
        {t("auth.mfa.add_device")}
      </Button>
      <Button variant="ghost" onClick={onReset}>
        <RefreshCw className="mr-2 h-4 w-4" />
        {t("auth.mfa.reset_all")}
      </Button>
    </div>
  );
}
