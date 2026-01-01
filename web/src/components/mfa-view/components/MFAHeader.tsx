import { useTranslation } from "react-i18next";
import { Shield, ShieldCheck } from "lucide-react";

import { CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useMFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";

export function MFAHeader() {
  const { t } = useTranslation();
  const { mfaEnabled, deviceCount, maxDevices } = useMFAViewContext();

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
