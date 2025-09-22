import { api } from "@/lib/api";

export interface IPasswordResetRequestData {
  email: string;
}

export interface IPasswordResetRequestResponse {
  message: string;
}

export interface IPasswordResetValidateData {
  code: string;
  new_password: string;
}

export interface IPasswordResetValidateResponse {
  access_token: string;
  refresh_token: string;
}

export const api_requestPasswordReset = (data: IPasswordResetRequestData) =>
  api.post<void>("/auth/reset-password", data);

export const api_validatePasswordReset = (
  challengeId: string,
  data: IPasswordResetValidateData,
) =>
  api.post<IPasswordResetValidateResponse>(
    `/auth/reset-password/${challengeId}/validate`,
    data,
  );
