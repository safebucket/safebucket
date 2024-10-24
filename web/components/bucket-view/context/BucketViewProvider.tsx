import React, { useState } from "react";

import { usePathname, useRouter } from "next/navigation";

import { BucketViewMode, IFile } from "@/components/bucket-view/helpers/types";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";

export const BucketViewProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const pathname = usePathname();
  const router = useRouter();

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
