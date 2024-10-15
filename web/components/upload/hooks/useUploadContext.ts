import { createContext, useContext } from "react";

import { IFileTransferContext } from "@/components/upload/helpers/types";

export const UploadContext = createContext<IFileTransferContext>(
  {} as IFileTransferContext,
);

export function useUploadContext() {
  const ctx = useContext(UploadContext);
  if (!ctx) {
    throw new Error("useUploadContext() called outside of context");
  }
  return ctx;
}
