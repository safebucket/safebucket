import React, { useState } from "react";

import { useNavigate, useParams } from "@tanstack/react-router";

import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { isFolder } from "@/components/bucket-view/helpers/utils";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import type { BucketItem } from "@/types/bucket.ts";

export const BucketViewProvider = ({
  children,
  folderId,
}: {
  children: React.ReactNode;
  folderId: string | null;
}) => {
  const params = useParams({ from: "/_authenticated/buckets/$id/$" });
  const navigate = useNavigate();

  const [view, setView] = useState<BucketViewMode>(BucketViewMode.List);
  const [selected, setSelected] = useState<BucketItem | null>(null);

  const openFolder = (item: BucketItem) => {
    if (isFolder(item)) {
      // Navigate to /buckets/{bucketId}/{folderId}
      navigate({
        to: "/buckets/$id/$",
        params: { id: params.id, _splat: item.id },
      });
    }
  };

  return (
    <BucketViewContext.Provider
      value={{
        bucketId: params.id,
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
