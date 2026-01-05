import type { IMFADevice } from "./types";

export function getDefaultDevice(
  devices: IMFADevice[],
): IMFADevice | undefined {
  const defaultDevice = devices.find((d) => d.is_default);
  return defaultDevice || devices[0];
}

export function getDefaultDeviceId(devices: IMFADevice[]): string {
  const defaultDevice = getDefaultDevice(devices);
  return defaultDevice?.id || "";
}

export function hasMultipleDevices(devices: IMFADevice[]): boolean {
  return devices.length > 1;
}

export function sortDevices(devices: IMFADevice[]): IMFADevice[] {
  return [...devices].sort((a, b) => {
    if (a.is_default && !b.is_default) return -1;
    if (!a.is_default && b.is_default) return 1;
    return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
  });
}

export function isDeviceDefault(device: IMFADevice): boolean {
  return device.is_default;
}

export function isDeviceVerified(device: IMFADevice): boolean {
  return device.is_verified;
}

export function isCodeValid(code: string): boolean {
  return code.length === 6 && /^\d+$/.test(code);
}

export function canAddDevice(deviceCount: number, maxDevices: number): boolean {
  return deviceCount < maxDevices;
}

export function getDeviceDisplayName(device: IMFADevice): string {
  return device.name;
}
