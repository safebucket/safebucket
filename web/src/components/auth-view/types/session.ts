import type {
  SubmitHandler,
  UseFormHandleSubmit,
  UseFormRegister,
  UseFormWatch,
} from "react-hook-form";

export type Status = "authenticated" | "loading" | "unauthenticated";

export type Session = {
  loggedUser: IUser | null;
  accessToken: string;
  refreshToken?: string;
  authProvider: string;
};

export interface IJWTPayload {
  aud: string;
  email: string;
  exp: number;
  iat: number;
  iss: string;
  user_id: string;
}

export interface IUser {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface ISessionContext {
  register: UseFormRegister<ILoginForm>;
  localLogin: SubmitHandler<ILoginForm>;
  handleSubmit: UseFormHandleSubmit<ILoginForm>;
  watch: UseFormWatch<ILoginForm>;

  login: (provider: string) => void;
  logout: () => void;
  setAuthenticationState: (
    accessToken: string,
    refreshToken: string,
    provider: string,
  ) => void;

  session: Session | null;
  status: string;
}

export interface ILoginForm {
  email: string;
  password: string;
}

export interface ILoginResponse {
  access_token: string;
  refresh_token: string;
}
