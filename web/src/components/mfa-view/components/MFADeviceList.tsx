import type { IMFADevice } from "@/components/mfa-view/helpers/types";
import { MFADeviceRow } from "@/components/mfa-view/components/MFADeviceRow";

interface MFADeviceListProps {
  devices: Array<IMFADevice>;
  onSetDefault: (deviceId: string) => void;
  onDelete: (deviceId: string) => void;
}

export function MFADeviceList({
  devices,
  onSetDefault,
  onDelete,
}: MFADeviceListProps) {
  return (
    <div className="space-y-3">
      {devices.map((device) => (
        <MFADeviceRow
          key={device.id}
          device={device}
          onSetDefault={onSetDefault}
          onDelete={onDelete}
        />
      ))}
    </div>
  );
}
