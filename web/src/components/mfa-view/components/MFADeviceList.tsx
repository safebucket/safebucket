import { useTranslation } from "react-i18next";

import { useMFAViewContext } from "@/components/mfa-view/hooks/useMFAViewContext";
import { MFADeviceRow } from "@/components/mfa-view/components/MFADeviceRow";

export function MFADeviceList() {
  const { t } = useTranslation();
  const { devices, isLoading } = useMFAViewContext();

  if (isLoading) {
    return (
      <div className="text-muted-foreground text-sm">{t("common.loading")}</div>
    );
  }

  if (devices.length === 0) {
    return null;
  }

  return (
    <div className="space-y-3">
      {devices.map((device) => (
        <MFADeviceRow key={device.id} device={device} />
      ))}
    </div>
  );
}
