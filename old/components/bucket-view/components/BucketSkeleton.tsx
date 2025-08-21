import React from "react";

import { Skeleton } from "@/components/ui/skeleton";

export const BucketSkeleton = () => {
  return (
    <>
      <div className="flex-1">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">
            <Skeleton className="h-10 w-[250px]" />
          </h1>
        </div>
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 2xl:grid-cols-6">
        <Skeleton className="h-[120px] w-[250px]" />
        <Skeleton className="h-[120px] w-[250px]" />
      </div>
    </>
  );
};
