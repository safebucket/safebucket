import { api } from "@/lib/api";

export interface IPasswordResetRequestData {
  email: string;
}

export interface IPasswordResetValidateData {
  code: string;
}

export interface IPasswordResetValidateResponse {
  mfa_required: boolean;
}

export interface IPasswordResetCompleteData {
  new_password: string;
}

export interface IMFAVerifyPasswordResetResponse {
  mfa_required: boolean;
}

export const api_requestPasswordReset = (data: IPasswordResetRequestData) =>
  api.post<void>("/auth/reset-password", data, { retryOnRateLimit: false });

export const api_validatePasswordReset = (
  challengeId: string,
  data: IPasswordResetValidateData,
) =>
  api.post<IPasswordResetValidateResponse>(
    `/auth/reset-password/${challengeId}/validate`,
    data,
    { retryOnRateLimit: false },
  );

export const api_completePasswordReset = (
  challengeId: string,
  data: IPasswordResetCompleteData,
) =>
  api.post<void>(`/auth/reset-password/${challengeId}/complete`, data, {
    retryOnRateLimit: false,
  });

export const api_verifyMFAPasswordReset = (code: string, deviceId?: string) =>
  api.post<IMFAVerifyPasswordResetResponse>(
    "/auth/mfa/verify",
    {
      code,
      ...(deviceId && { device_id: deviceId }),
    },
    { retryOnRateLimit: false },
  );
