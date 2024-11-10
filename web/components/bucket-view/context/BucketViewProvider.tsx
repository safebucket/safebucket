import React, { useState } from "react";

import { useParams, usePathname, useRouter } from "next/navigation";

import { BucketViewMode, IFile } from "@/components/bucket-view/helpers/types";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";

export const BucketViewProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const pathname = usePathname();
  const router = useRouter();
  const params = useParams<{ id: string }>();

  const [view, setView] = useState<BucketViewMode>(BucketViewMode.List);
  const [selected, setSelected] = useState<IFile | null>(null);

  const openFolder = (file: IFile) => {
    if (file.type == "folder") {
      router.push(`${pathname}/${file.name}`);
    }
  };

  return (
    <BucketViewContext.Provider
      value={{
        bucketId: params.id,
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
