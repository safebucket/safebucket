export interface IPasswordResetRequestFormData {
  email: string;
}

// Password entry step - new password
export interface IPasswordResetPasswordFormData {
  newPassword: string;
  confirmPassword: string;
}

// Stage of the password reset flow
export type PasswordResetStage = "code" | "mfa" | "password" | "success";
