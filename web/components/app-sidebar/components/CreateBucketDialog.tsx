import React, { FC } from "react";

import { Plus } from "lucide-react";

import { useBucketsData } from "@/components/bucket-view/hooks/useBucketsData";
import { CustomDialog } from "@/components/common/components/CustomDialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export const CreateBucketDialog: FC = () => {
  const {
    isDialogOpen,
    setIsDialogOpen,
    createBucket,
    register,
    handleSubmit,
  } = useBucketsData();

  return (
    <CustomDialog
      title="New bucket"
      description="Create a bucket to share files safely"
      trigger={<Plus />}
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
  );
};
