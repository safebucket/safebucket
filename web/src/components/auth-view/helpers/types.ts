export interface IPasswordResetRequestFormData {
  email: string;
}

export interface IPasswordResetValidateFormData {
  code: string;
  newPassword: string;
  confirmPassword: string;
}
