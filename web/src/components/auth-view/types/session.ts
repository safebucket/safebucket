export type Session = {
  userId: string;
  email: string;
  role: "admin" | "user" | "guest";
  authProvider: string;
};

export interface IJWTPayload {
  aud: string;
  email: string;
  exp: number;
  iat: number;
  iss: string;
  user_id: string;
  role: "admin" | "user" | "guest";
}

export interface IUser {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  provider_type: string;
  role: "admin" | "user" | "guest";
  mfa_enabled: boolean;
  mfa_enabled_at?: string;
  created_at: string;
  updated_at: string;
}

export interface ILoginForm {
  email: string;
  password: string;
}

export type MFADeviceType = "totp";

export interface IMFADevice {
  id: string;
  name: string;
  type: MFADeviceType;
  is_default: boolean;
  created_at: string;
  verified_at?: string;
  last_used_at?: string;
}

export interface IMFADevicesResponse {
  devices: Array<IMFADevice>;
  mfa_enabled: boolean;
  device_count: number;
  max_devices: number;
}

export interface IMFADeviceSetupResponse {
  device_id: string;
  secret: string;
  qr_code_uri: string;
  issuer: string;
}

export interface ILoginResponse {
  access_token?: string;
  refresh_token?: string;
  mfa_required: boolean;
  mfa_token?: string;
  mfa_setup_required?: boolean;
}
