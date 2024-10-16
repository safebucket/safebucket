import { createContext, useContext } from "react";

import { ISessionContext } from "@/components/auth-view/types/session";

export const SessionContext = createContext<ISessionContext>({} as ISessionContext);

export function useSessionContext() {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSessionContext() called outside of context");
  }
  return ctx;
}
