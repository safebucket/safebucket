import {
  SubmitHandler,
  UseFormHandleSubmit,
  UseFormRegister,
} from "react-hook-form";

export type Status = "authenticated" | "loading" | "unauthenticated";

export type Session = {
  accessToken: string;
  refreshToken?: string;
  authProvider: string;
};

export interface ISessionContext {
  register: UseFormRegister<ILoginForm>;
  localLogin: SubmitHandler<ILoginForm>;
  handleSubmit: UseFormHandleSubmit<ILoginForm>;
  login(provider: string): void;
  logout(): void;
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
