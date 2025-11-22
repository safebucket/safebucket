import { createContext, useContext } from "react";

import type { BucketViewMode } from "@/components/bucket-view/helpers/types";
import type { BucketItem } from "@/types/bucket.ts";

export interface IBucketViewContext {
  bucketId: string;
  folderId: string | null;
  view: BucketViewMode;
  setView: (view: BucketViewMode) => void;
  selected: BucketItem | null;
  setSelected: (item: BucketItem) => void;
  openFolder: (item: BucketItem) => void;
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
