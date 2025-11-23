import { createContext, useContext } from "react";

import type { IUploadContext } from "@/components/upload/helpers/types";

export const UploadContext = createContext<IUploadContext | null>(null);

export function useUploadContext() {
  const ctx = useContext(UploadContext);
  if (!ctx) {
    throw new Error("useUploadContext must be used within UploadProvider");
  }
  return ctx;
}
