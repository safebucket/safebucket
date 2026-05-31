import { createContext, useContext } from "react";

import type { OnChangeFn, RowSelectionState } from "@tanstack/react-table";
import type { BucketViewMode } from "@/components/bucket-view/helpers/types";
import type { BucketItem } from "@/types/bucket.ts";

export interface IBucketViewContext {
  bucketId: string;
  folderId: string | undefined;
  view: BucketViewMode;
  setView: (view: BucketViewMode) => void;
  selected: BucketItem | null;
  setSelected: (item: BucketItem) => void;
  openItem: (item: BucketItem) => void;
  previewItem: BucketItem | null;
  closePreview: () => void;
  rowSelection: RowSelectionState;
  setRowSelection: OnChangeFn<RowSelectionState>;
  clearRowSelection: () => void;
}

export const BucketViewContext = createContext<IBucketViewContext | null>(null);

export function useBucketViewContext() {
  const ctx = useContext(BucketViewContext);
  if (ctx === null) {
    throw new Error("useBucketViewContext() called outside of context");
  }
  return ctx;
}
