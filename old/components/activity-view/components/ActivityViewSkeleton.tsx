import React from "react";

import { Separator } from "@radix-ui/react-menu";

import { Skeleton } from "@/components/ui/skeleton";

export const ActivityViewSkeleton = () => {
  return (
    <>
      {Array.from({ length: 5 }).map((_, index) => (
        <div key={index}>
          <div className="flex items-start space-x-4 py-4">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="flex-1 space-y-2">
              <div className="flex items-center space-x-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-6 w-6 rounded-full" />
              </div>
              <Skeleton className="h-4 w-full max-w-md" />
              <Skeleton className="h-3 w-20" />
            </div>
          </div>
          {index < 4 && <Separator />}
        </div>
      ))}
    </>
  );
};
