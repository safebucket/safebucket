export interface IPasswordResetRequestFormData {
  email: string;
}

export interface IPasswordResetPasswordFormData {
  newPassword: string;
  confirmPassword: string;
}

export type PasswordResetStage = "code" | "mfa" | "password" | "success";
