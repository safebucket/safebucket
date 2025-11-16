export type Status = "authenticated" | "loading" | "unauthenticated";

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
  created_at: string;
  updated_at: string;
}

export interface ISessionContext {
  login: (provider: string) => void;
  logout: () => void;
  setAuthenticationState: (
    accessToken: string,
    refreshToken: string,
    provider: string,
  ) => void;
  refreshSession: () => void;

  session: Session | null;
  status: Status;
}

export interface ILoginForm {
  email: string;
  password: string;
}

export interface ILoginResponse {
  access_token: string;
  refresh_token: string;
}
