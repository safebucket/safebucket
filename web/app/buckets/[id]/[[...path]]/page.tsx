"use client";

import React from "react";

import { BucketView } from "@/components/bucket-view/BucketView";
import { BucketSkeleton } from "@/components/bucket-view/components/BucketSkeleton";
import { useBucketData } from "@/components/bucket-view/hooks/useBucketData";

interface IBucketProps {
  params: { id: string; path?: string[] };
}

export default function Bucket({ params }: IBucketProps) {
  const { bucket, isLoading } = useBucketData(params.id);

  const path = params.path ? `/${params.path.join("/")}` : "/";

  return (
    <div className="flex-1">
      <div className="m-6 mt-0 grid grid-cols-1 gap-8">
        {isLoading ? (
          <BucketSkeleton />
        ) : (
          <BucketView bucket={bucket!} path={path} />
        )}
      </div>
    </div>
  );
}
