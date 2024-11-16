import React, { FC, useState } from "react";

import { PlusCircle } from "lucide-react";

import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { shareFileFields } from "@/components/bucket-view/helpers/constants";
import { IBucket } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { UploadPopover } from "@/components/upload/components/UploadPopover";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";

interface IBucketHeaderProps {
  bucket: IBucket;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
}: IBucketHeaderProps) => {
  const [filterType, setFilterType] = useState("all");
  const shareFileDialog = useDialog();

  const { path } = useBucketViewContext();
  const { startUpload } = useUploadContext();

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

          <Button onClick={shareFileDialog.trigger}>
            <PlusCircle className="mr-2 h-4 w-4" />
            Share a file
          </Button>

          <FormDialog
            {...shareFileDialog.props}
            title="Share a file"
            description="Upload a file and share it safely"
            fields={shareFileFields}
            onSubmit={(data) => startUpload(data.files, path, bucket.id)}
            confirmLabel="Share"
          />
        </div>
      </div>
    </div>
  );
};
