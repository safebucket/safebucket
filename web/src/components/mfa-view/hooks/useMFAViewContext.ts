import { createContext, useContext } from "react";

import type { IMFAViewContext } from "@/components/mfa-view/helpers/types";

export const MFAViewContext = createContext<IMFAViewContext | null>(null);

export function useMFAViewContext(): IMFAViewContext {
  const ctx = useContext(MFAViewContext);
  if (!ctx) {
    throw new Error("useMFAViewContext must be used within MFAViewProvider");
  }
  return ctx;
}
