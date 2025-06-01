"use client";

import React, { use } from "react";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketSkeleton } from "@/components/bucket-view/components/BucketSkeleton";
import { BucketViewProvider } from "@/components/bucket-view/context/BucketViewProvider";
import { useBucketData } from "@/components/bucket-view/hooks/useBucketData";

interface IBucketProps {
  params: Promise<{ id: string; path?: string[] }>;
}

export default function Bucket(props: IBucketProps) {
  const params = use(props.params);
  const { bucket, isLoading } = useBucketData(params.id);

  const path = params.path ? `/${params.path.join("/")}` : "/";

  return (
    <BucketViewProvider path={path}>
      <div className="flex-1 w-full overflow-auto">
        <div className="m-6 mt-0 grid grid-cols-1 gap-8">
          {isLoading ? (
            <BucketSkeleton />
          ) : (
            <BucketView bucket={bucket!} />
          )}
        </div>
      </div>
    </BucketViewProvider>
  );
}
