export interface IPasswordResetRequestFormData {
  email: string;
}

// Code verification step - code only
export interface IPasswordResetCodeFormData {
  code: string;
}

// Password entry step - new password
export interface IPasswordResetPasswordFormData {
  newPassword: string;
  confirmPassword: string;
}

// Stage of the password reset flow
export type PasswordResetStage = "code" | "mfa" | "password" | "success";

// Legacy type for backwards compatibility
export interface IPasswordResetValidateFormData {
  code: string;
  newPassword: string;
  confirmPassword: string;
}
