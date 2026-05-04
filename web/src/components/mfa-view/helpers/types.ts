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
