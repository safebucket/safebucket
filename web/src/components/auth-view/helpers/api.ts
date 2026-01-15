import { api } from "@/lib/api";

export interface IPasswordResetRequestData {
  email: string;
}

// Code verification only (no password in this step)
export interface IPasswordResetValidateData {
  code: string;
}

// Response from code verification - returns restricted access token
export interface IPasswordResetValidateResponse {
  access_token: string; // Restricted access token for password reset flow
  mfa_required: boolean;
}

// Password reset completion request (token in Authorization header)
export interface IPasswordResetCompleteData {
  new_password: string;
}

// Response from password reset completion - returns full auth tokens
export interface IPasswordResetCompleteResponse {
  access_token: string;
  refresh_token: string;
}

// Response from MFA verification during password reset
// Returns upgraded restricted token with MFA=true
export interface IMFAVerifyPasswordResetResponse {
  access_token: string; // Upgraded restricted token
  mfa_required: boolean;
}

export const api_requestPasswordReset = (data: IPasswordResetRequestData) =>
  api.post<void>("/auth/reset-password", data);

// Validate reset code - returns restricted access token
export const api_validatePasswordReset = (
  challengeId: string,
  data: IPasswordResetValidateData,
) =>
  api.post<IPasswordResetValidateResponse>(
    `/auth/reset-password/${challengeId}/validate`,
    data,
  );

// Complete password reset (requires restricted access token in header)
export const api_completePasswordReset = (
  challengeId: string,
  data: IPasswordResetCompleteData,
  restrictedToken: string,
) =>
  api.post<IPasswordResetCompleteResponse>(
    `/auth/reset-password/${challengeId}/complete`,
    data,
    { headers: { Authorization: `Bearer ${restrictedToken}` } },
  );

// Verify MFA code for password reset flow (requires restricted token in header)
export const api_verifyMFAPasswordReset = (
  restrictedToken: string,
  code: string,
  deviceId?: string,
) =>
  api.post<IMFAVerifyPasswordResetResponse>(
    "/auth/mfa/verify",
    {
      code,
      ...(deviceId && { device_id: deviceId }),
    },
    { headers: { Authorization: `Bearer ${restrictedToken}` } },
  );
