import { useTranslation } from "react-i18next";
import { Shield, ShieldCheck } from "lucide-react";

import { CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

interface MFAHeaderProps {
  mfaEnabled: boolean;
  deviceCount: number;
  maxDevices: number;
}

export function MFAHeader({
  mfaEnabled,
  deviceCount,
  maxDevices,
}: MFAHeaderProps) {
  const { t } = useTranslation();

  return (
    <CardHeader>
      <CardTitle className="flex items-center gap-2 text-base">
        {mfaEnabled ? (
          <ShieldCheck className="h-4 w-4 text-green-600" />
        ) : (
          <Shield className="h-4 w-4" />
        )}
        {t("auth.mfa.devices_title")}
      </CardTitle>
      <CardDescription>
        {t("auth.mfa.devices_description", {
          count: deviceCount,
          max: maxDevices,
        })}
      </CardDescription>
    </CardHeader>
  );
}
