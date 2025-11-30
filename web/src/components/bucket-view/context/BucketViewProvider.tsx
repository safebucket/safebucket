import React, { useState } from "react";

import { useNavigate, useParams } from "@tanstack/react-router";

import type { BucketItem } from "@/types/bucket.ts";
import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";

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

  const openFolder = (item: BucketItem) => {
    if (isFolder(item)) {
      // Navigate to /buckets/{bucketId}/{folderId}
      navigate({ to: `/buckets/${params.bucketId}/${item.id}` });
    }
  };

  return (
    <BucketViewContext.Provider
      value={{
        bucketId: params.bucketId,
        folderId,
        view,
        setView,
        selected,
        setSelected,
        openFolder,
      }}
    >
      {children}
    </BucketViewContext.Provider>
  );
};
