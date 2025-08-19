export type Status = "authenticated" | "loading" | "unauthenticated";

export interface User {
  id: string;
  first_name: string;
  last_name: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface Session {
  loggedUser: User | null;
  accessToken: string;
  refreshToken?: string;
  authProvider: string;
}

export interface SessionContextType {
  session: Session | null;
  status: Status;
  logout: () => void;
}
