import React, { FC } from "react";

import { IBucket } from "@/components/bucket-view/helpers/types";
import { FileItemView } from "@/components/common/components/FileItemView";
import { Skeleton } from "@/components/ui/skeleton";

interface IBucketContentProps {
  bucket: IBucket | undefined;
  isLoading: boolean;
}

export const BucketContent: FC<IBucketContentProps> = ({
  bucket,
  isLoading,
}: IBucketContentProps) => {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
      {isLoading && <Skeleton className="h-[120px] w-[250px]" />}
      {!isLoading &&
        bucket!.files.map((file) => <FileItemView key={file.id} file={file} />)}
    </div>
  );
};
