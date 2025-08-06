import React, { useState } from "react";

import { useParams, usePathname, useRouter } from "next/navigation";

import { BucketViewMode, IFile } from "@/components/bucket-view/helpers/types";
import { BucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { successToast } from "@/components/ui/hooks/use-toast";
import { api_deleteBucket } from "@/components/upload/helpers/api";

export const BucketViewProvider = ({
  children,
  path,
}: {
  children: React.ReactNode;
  path: string;
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

  const addMembers = (bucketId: string) => {};

  const deleteBucket = (bucketId: string) =>
    api_deleteBucket(bucketId).then(() =>
      successToast(`Bucket ${bucketId} has been deleted`),
    );

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
        addMembers,
        deleteBucket,
      }}
    >
      {children}
    </BucketViewContext.Provider>
  );
};
