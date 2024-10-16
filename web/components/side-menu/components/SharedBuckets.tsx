import React, { FC } from "react";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketsData } from "@/components/bucket-view/hooks/useBucketsData";

import { CustomDialog } from "@/components/common/components/CustomDialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";

export const SharedBuckets: FC = () => {
  const pathname = usePathname();
  const {
    buckets,
    isLoading,
    isDialogOpen,
    setIsDialogOpen,
    createBucket,
    register,
    handleSubmit,
  } = useBucketsData();

  return (
    <>
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium">Shared buckets</h3>
        <CustomDialog
          title="New bucket"
          description="Create a bucket to share files safely"
          trigger={
            <Button
              variant="outline"
              size="sm"
              className="hover:bg-muted hover:text-primary"
            >
              New
            </Button>
          }
          submitName="Create"
          onSubmit={handleSubmit(createBucket)}
          isOpen={isDialogOpen}
          setIsOpen={setIsDialogOpen}
        >
          <div className="grid grid-cols-4 items-center gap-4">
            <Label htmlFor="name">Name</Label>
            <Input
              id="name"
              type="name"
              {...register("name", { required: true })}
              className="col-span-3"
            />
          </div>
        </CustomDialog>
      </div>
      <nav className="space-y-1">
        {isLoading && <Skeleton className="h-10" />}
        {!isLoading &&
          buckets.map((bucket: IBucket) => (
            <Link
              key={bucket.id}
              href={`/buckets/${bucket.id}`}
              className={`block rounded-md px-3 py-2 hover:bg-muted ${pathname == `/buckets/${bucket.id}` ? "bg-muted text-primary" : ""}`}
            >
              {bucket.name}
            </Link>
          ))}
      </nav>
    </>
  );
};
