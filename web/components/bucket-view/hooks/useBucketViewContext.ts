import { createContext, useContext } from "react";

import { BucketViewMode, IFile } from "@/components/bucket-view/helpers/types";

export interface IBucketViewContext {
  bucketId: string;
  view: BucketViewMode;
  setView: (view: BucketViewMode) => void;
  selected: IFile | null;
  setSelected: (file: IFile) => void;
  openFolder: (file: IFile) => void;
}

export const BucketViewContext = createContext<IBucketViewContext>(
  {} as IBucketViewContext,
);

export function useBucketViewContext() {
  const ctx = useContext(BucketViewContext);
  if (!ctx) {
    throw new Error("useBucketViewContext() called outside of context");
  }
  return ctx;
}
