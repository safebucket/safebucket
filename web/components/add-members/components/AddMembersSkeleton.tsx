import React, { FC } from "react";

import { Skeleton } from "@/components/ui/skeleton";

export const AddMembersSkeleton: FC = () => {
  return (
    <div className="mb-2 space-y-4">
      <div className="text-sm font-medium">People with access</div>
      
      {/* Skeleton for current user */}
      <div className="flex items-center justify-between space-x-4">
        <div className="flex items-center space-x-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-32" />
          </div>
        </div>
        <Skeleton className="h-8 w-16" />
      </div>
      
      {/* Skeleton for 2-3 members */}
      {[...Array(2)].map((_, index) => (
        <div key={index} className="mb-2 grid grid-cols-12 items-center">
          <div className="col-span-8 flex items-center space-x-4">
            <Skeleton className="h-10 w-10 rounded-full" />
            <div className="space-y-2">
              <Skeleton className="h-4 w-28" />
              <Skeleton className="h-3 w-36" />
            </div>
          </div>
          <div className="col-span-3 mr-1 flex">
            <Skeleton className="h-8 w-20 ml-auto" />
          </div>
          <div className="col-span-1">
            <Skeleton className="h-8 w-8" />
          </div>
        </div>
      ))}
    </div>
  );
};