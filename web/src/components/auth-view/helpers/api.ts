import { api } from "@/lib/api";
import type { IMFADevice } from "@/components/mfa-view/helpers/types";

export interface IPasswordResetRequestData {
  email: string;
}

export interface IPasswordResetRequestResponse {
  message: string;
}

// Code verification only (no password in this step)
export interface IPasswordResetValidateData {
  code: string;
}

// Response from code verification - may require MFA or return completion token
export interface IPasswordResetValidateResponse {
  mfa_required: boolean;
  mfa_token?: string;
  devices?: IMFADevice[];
  completion_token?: string;
}

// Password reset completion request (after code + MFA verification)
export interface IPasswordResetCompleteData {
  completion_token: string;
  new_password: string;
}

// Response from password reset completion - returns auth tokens
export interface IPasswordResetCompleteResponse {
  access_token: string;
  refresh_token: string;
}

// Response from MFA verification during password reset
export interface IMFAVerifyPasswordResetResponse {
  completion_token?: string;
  password_reset?: boolean;
  access_token?: string;
  refresh_token?: string;
}

export const api_requestPasswordReset = (data: IPasswordResetRequestData) =>
  api.post<void>("/auth/reset-password", data);

// Validate reset code - returns MFA token or completion token
export const api_validatePasswordReset = (
  challengeId: string,
  data: IPasswordResetValidateData,
) =>
  api.post<IPasswordResetValidateResponse>(
    `/auth/reset-password/${challengeId}/validate`,
    data,
  );

// Complete password reset with completion token
export const api_completePasswordReset = (
  challengeId: string,
  data: IPasswordResetCompleteData,
) =>
  api.post<IPasswordResetCompleteResponse>(
    `/auth/reset-password/${challengeId}/complete`,
    data,
  );

// Verify MFA code for password reset flow
export const api_verifyMFAPasswordReset = (
  mfaToken: string,
  code: string,
  deviceId?: string,
) =>
  api.post<IMFAVerifyPasswordResetResponse>("/auth/mfa/verify", {
    mfa_token: mfaToken,
    code,
    ...(deviceId && { device_id: deviceId }),
  });
