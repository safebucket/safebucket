import { createContext, useContext } from "react";

import type { ISessionContext } from "@/components/auth-view/types/session";

export const SessionContext = createContext<ISessionContext | null>(null);

export function useSessionContext() {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSessionContext must be used within SessionProvider");
  }
  return ctx;
}
