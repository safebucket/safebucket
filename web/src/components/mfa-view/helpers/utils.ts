import type { IMFADevice } from "./types";

export function getDefaultDevice(
  devices: Array<IMFADevice>,
): IMFADevice | undefined {
  const defaultDevice = devices.find((d) => d.is_default);
  return defaultDevice || devices[0];
}

export function getDefaultDeviceId(devices: Array<IMFADevice>): string {
  const defaultDevice = getDefaultDevice(devices);
  return defaultDevice?.id || "";
}

export function hasMultipleDevices(devices: Array<IMFADevice>): boolean {
  return devices.length > 1;
}

export function isCodeValid(code: string): boolean {
  return code.length === 6 && /^\d+$/.test(code);
}
