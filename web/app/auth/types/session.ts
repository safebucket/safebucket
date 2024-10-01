export type Status = "authenticated" | "loading" | "unauthenticated";

export type Session = {
  accessToken: string;
  refreshToken?: string;
  authProvider: string;
};

export interface ISessionContext {
  login(provider: string): void;
  logout(): void;
  session: Session | null;
  status: string;
}
