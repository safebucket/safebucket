import React, { FC, useState } from "react";

import { ChevronDown, PlusCircle, Trash2, UserPlus } from "lucide-react";

import { AddMembers } from "@/components/add-members";
import { useSessionContext } from "@/components/auth-view/hooks/useSessionContext";
import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { shareFileFields } from "@/components/bucket-view/helpers/constants";
import { IBucket, IInvites } from "@/components/bucket-view/helpers/types";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
  const [shareWith, setShareWith] = useState<IInvites[]>([]);

  const shareFileDialog = useDialog();
  const addMembersDialog = useDialog();
  const deleteBucketDialog = useDialog();

  const { session } = useSessionContext();
  const { path, deleteBucket } = useBucketViewContext();
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

          <div className="flex items-center">
            <Button
              onClick={shareFileDialog.trigger}
              className="rounded-r-none"
            >
              <PlusCircle className="mr-2 h-4 w-4" />
              Share a file
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="default" className="rounded-l-none border-l-0">
                  <ChevronDown className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={addMembersDialog.trigger}>
                  <UserPlus className="mr-2 h-4 w-4" />
                  Add members
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  className="text-red-600"
                  onClick={deleteBucketDialog.trigger}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete bucket
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>

          <FormDialog
            {...shareFileDialog.props}
            title="Share a file"
            description="Upload a file and share it safely"
            fields={shareFileFields}
            onSubmit={(data) => startUpload(data.files, path, bucket.id)}
            confirmLabel="Share"
          />

          <FormDialog
            {...addMembersDialog.props}
            maxWidth="650px"
            title="Add members"
            description="Share this bucket with team members"
            fields={[]}
            onSubmit={() => {
              // TODO: Implement actual member adding logic
              console.log("Adding members:", shareWith);
              setShareWith([]);
            }}
            confirmLabel="Add members"
          >
            <AddMembers
              shareWith={shareWith}
              onShareWithChange={setShareWith}
              currentUserEmail={session?.loggedUser?.email}
              currentUserName={`${session?.loggedUser?.first_name} ${session?.loggedUser?.last_name}`}
              bucketId={bucket?.id}
            />
          </FormDialog>

          <CustomAlertDialog
            {...deleteBucketDialog.props}
            title={`Delete ${bucket.name}?`}
            description="Are you sure you want to delete this bucket? This action cannot be undone."
            confirmLabel="Delete"
            onConfirm={() => deleteBucket(bucket.id)}
          />
        </div>
      </div>
    </div>
  );
};
