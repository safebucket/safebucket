import React, { FC } from "react";

import { Bucket } from "@/app/buckets/helpers/types";

import { FileView } from "@/components/fileview";
import { Skeleton } from "@/components/ui/skeleton";

interface IBucketContentProps {
  bucket: Bucket | undefined;
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
        bucket!.files.map((file) => <FileView key={file.id} file={file} />)}
    </div>
  );
};
