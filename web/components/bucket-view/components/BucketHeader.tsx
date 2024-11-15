import React, { FC, useState } from "react";

import { PlusCircle } from "lucide-react";
import { useForm } from "react-hook-form";

import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { CustomDialog } from "@/components/common/components/CustomDialog";
import { Datepicker } from "@/components/common/components/Datepicker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { UploadPopover } from "@/components/upload/components/UploadPopover";
import { IStartUploadData } from "@/components/upload/helpers/types";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

interface IBucketHeaderProps {
  bucket: IBucket;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
}: IBucketHeaderProps) => {
  const [filterType, setFilterType] = useState("all");
  const [expiresAt, setExpiresAt] = useState(false);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const { path } = useBucketViewContext();
  const { startUpload } = useUploadContext();

  const { register, handleSubmit } = useForm<IStartUploadData>();

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{bucket.name}</h1>
        <div className="flex items-center gap-4">
          <BucketViewOptions />

          <Select value={filterType} onValueChange={setFilterType}>
            <SelectTrigger>
              <SelectValue placeholder="Filter by type" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="pdf">PDF</SelectItem>
              <SelectItem value="pptx">PowerPoint</SelectItem>
              <SelectItem value="jpg">Image</SelectItem>
              <SelectItem value="xlsx">Excel</SelectItem>
              <SelectItem value="mp4">Video</SelectItem>
              <SelectItem value="mp3">Audio</SelectItem>
            </SelectContent>
          </Select>

          <UploadPopover />

          <CustomDialog
            title="Share a file"
            description="Upload a file and share it safely"
            trigger={
              <Button>
                <PlusCircle className="mr-2 h-4 w-4" />
                Share a file
              </Button>
            }
            submitName="Share"
            onSubmit={handleSubmit((data) => {
              setIsDialogOpen(false);
              startUpload(data, path, bucket.id);
            })}
            isOpen={isDialogOpen}
            setIsOpen={setIsDialogOpen}
          >
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="file" className="">
                File
              </Label>
              <Input
                id="files"
                type="file"
                className="col-span-3"
                {...register("files", { required: true })}
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="username" className="">
                Password
              </Label>
              <Input
                id="username"
                defaultValue="0UymxETG$wc)7k8"
                className="col-span-3"
              />
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="max-downloads" className="">
                Max downloads
              </Label>
              <Select>
                <SelectTrigger id="max-downloads" className="col-span-3">
                  <SelectValue
                    placeholder="Unlimited"
                    defaultValue="unlimited"
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="unlimited">Unlimited</SelectItem>
                  <SelectItem value="1">1</SelectItem>
                  <SelectItem value="3">3</SelectItem>
                  <SelectItem value="5">5</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid grid-cols-4 items-center gap-4">
              <Label htmlFor="expires-at" className="">
                Expires at
              </Label>
              <Switch
                id="expires-at"
                checked={expiresAt}
                onCheckedChange={setExpiresAt}
              />
            </div>
            {expiresAt && (
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="expires-at-date" className="">
                  Date
                </Label>
                <Datepicker />
              </div>
            )}
          </CustomDialog>
        </div>
      </div>
    </div>
  );
};
