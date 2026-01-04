export interface IMFADevice {
  id: string;
  name: string;
  type: "totp";
  is_default: boolean;
  is_verified: boolean;
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
  mfa_enabled: boolean;
  device_count: number;
  max_devices: number;
}

export interface IMFAResetRequestResponse {
  challenge_id: string;
}

export type SetupStep = "name" | "qr" | "verify" | "success";
export type ResetStep = "password" | "email_sent" | "success";

export type VerificationViewMode = "form" | "success";
export type SetupRequiredViewMode =
  | "loading"
  | "error"
  | "name"
  | "qr"
  | "verify"
  | "success";

export interface IMFAViewContext {
  userId: string;

  devices: Array<IMFADevice>;
  isLoading: boolean;
  mfaEnabled: boolean;
  deviceCount: number;
  maxDevices: number;

  setupDialogOpen: boolean;
  deleteDeviceId: string | null;
  resetDialogOpen: boolean;

  openSetupDialog: () => void;
  openDeleteDialog: (deviceId: string) => void;
  openResetDialog: () => void;
  closeAllDialogs: () => void;

  setDeviceDefault: (deviceId: string) => void;
}

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

export interface ISetupRequiredFlowState {
  viewMode: SetupRequiredViewMode;
  sessionError: boolean;
  handleSuccess: () => void;
  handleError: () => void;
  handleRetry: () => void;
}
