import React, { useState } from "react";

import { useLocation, useParams, useRouter } from "@tanstack/react-router";

import type { IFile } from "@/components/bucket-view/helpers/types";
import { BucketViewMode } from "@/components/bucket-view/helpers/types";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";

export const BucketViewProvider = ({
  children,
  path,
}: {
  children: React.ReactNode;
  path: string;
}) => {
  const router = useRouter();
  const params = useParams({ from: "/buckets/$id/" });
  const location = useLocation();

  const [view, setView] = useState<BucketViewMode>(BucketViewMode.List);
  const [selected, setSelected] = useState<IFile | null>(null);

  const openFolder = (file: IFile) => {
    if (file.type == "folder") {
      router.navigate({ to: `${location.pathname}/${file.name}` });
    }
  };

  return (
    <BucketViewContext.Provider
      value={{
        bucketId: params.id,
        path,
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
