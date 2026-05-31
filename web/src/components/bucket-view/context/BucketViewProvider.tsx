import React, { useCallback, useEffect, useState } from "react";

import { useNavigate, useParams } from "@tanstack/react-router";

import type { RowSelectionState } from "@tanstack/react-table";
import type { BucketItem } from "@/types/bucket.ts";
import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { isFile, isFolder } from "@/components/bucket-view/helpers/utils";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FilePreviewMount } from "@/components/bucket-view/components/FilePreviewMount.tsx";

export const BucketViewProvider = ({
  children,
  folderId,
}: {
  children: React.ReactNode;
  folderId: string | undefined;
}) => {
  const params = useParams({
    from: "/_authenticated/buckets/$bucketId/{-$folderId}",
  });
  const navigate = useNavigate();

  const [view, setView] = useState<BucketViewMode>(BucketViewMode.List);
  const [selected, setSelected] = useState<BucketItem | null>(null);
  const [previewItem, setPreviewItem] = useState<BucketItem | null>(null);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const clearRowSelection = useCallback(() => setRowSelection({}), []);

  useEffect(() => {
    setRowSelection({});
  }, [folderId, view]);

  const openItem = (item: BucketItem) => {
    if (isFolder(item)) {
      navigate({ to: `/buckets/${params.bucketId}/${item.id}` });
      return;
    }
    if (isFile(item)) {
      setPreviewItem(item);
    }
  };

  const closePreview = () => setPreviewItem(null);

  return (
    <BucketViewContext.Provider
      value={{
        bucketId: params.bucketId,
        folderId,
        view,
        setView,
        selected,
        setSelected,
        openItem,
        previewItem,
        closePreview,
        rowSelection,
        setRowSelection,
        clearRowSelection,
      }}
    >
      {children}
      <FilePreviewMount />
    </BucketViewContext.Provider>
  );
};
