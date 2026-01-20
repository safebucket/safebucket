export interface IMFADevice {
  id: string;
  name: string;
  type: "totp";
  is_default: boolean;
  created_at: string;
  verified_at?: string;
  last_used_at?: string;
}

export interface IMFADeviceSetupResponse {
  device_id: string;
  secret: string;
  qr_code_uri: string;
  issuer: string;
}

export interface IMFADevicesResponse {
  devices: Array<IMFADevice>;
}

export type SetupStep = "name" | "qr" | "verify" | "success";

export interface IVerificationFlowState {
  code: string;
  setCode: (code: string) => void;
  selectedDeviceId: string;
  setSelectedDeviceId: (deviceId: string) => void;
  error: string | null;
  isLoading: boolean;
  isVerified: boolean;
  handleSubmit: (e: React.FormEvent) => Promise<void>;
  handleBackToLogin: () => void;
}
