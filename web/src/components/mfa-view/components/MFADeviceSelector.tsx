import { useTranslation } from "react-i18next";
import { Smartphone, Star } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import type { IMFADevice } from "../helpers/types";
import { hasMultipleDevices } from "../helpers/utils";

export interface IMFADeviceSelectorProps {
  devices: IMFADevice[];
  selectedDeviceId: string;
  onSelectDevice: (deviceId: string) => void;
  disabled?: boolean;
}

export function MFADeviceSelector({
  devices,
  selectedDeviceId,
  onSelectDevice,
  disabled = false,
}: IMFADeviceSelectorProps) {
  const { t } = useTranslation();

  if (!hasMultipleDevices(devices)) {
    return null;
  }

  return (
    <div className="space-y-2">
      <Label htmlFor="device-select">{t("auth.mfa.select_device")}</Label>
      <Select
        value={selectedDeviceId}
        onValueChange={onSelectDevice}
        disabled={disabled}
      >
        <SelectTrigger id="device-select" className="w-full">
          <SelectValue placeholder={t("auth.mfa.select_device")} />
        </SelectTrigger>
        <SelectContent>
          {devices.map((device) => (
            <SelectItem key={device.id} value={device.id}>
              <div className="flex items-center gap-2">
                <Smartphone className="h-4 w-4" />
                <span>{device.name}</span>
                {device.is_default && (
                  <span className="ml-1 flex items-center gap-1 text-xs text-muted-foreground">
                    <Star className="h-3 w-3 fill-current" />
                    {t("auth.mfa.device_default")}
                  </span>
                )}
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
