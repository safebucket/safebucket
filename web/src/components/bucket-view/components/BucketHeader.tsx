import { useTranslation } from "react-i18next";
import { ChevronDownIcon, FolderPlus, PlusCircle } from "lucide-react";
import type { FC } from "react";

import type { IBucket } from "@/types/bucket.ts";
import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { shareFileFields } from "@/components/bucket-view/helpers/constants";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { Button } from "@/components/ui/button";
import { UploadPopover } from "@/components/upload/components/UploadPopover";
import { useUploadContext } from "@/components/upload/hooks/useUploadContext";
import { ButtonGroup } from "@/components/ui/button-group.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { useFileActions } from "@/components/FileActions/hooks/useFileActions.ts";

interface IBucketHeaderProps {
  bucket: IBucket;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
}: IBucketHeaderProps) => {
  const { t } = useTranslation();
  const shareFileDialog = useDialog();
  const newFolderDialog = useDialog();

  const { folderId } = useBucketViewContext();
  const { startUpload } = useUploadContext();
  const { createFolder } = useFileActions();

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{bucket.name}</h1>
        <div className="flex items-center gap-4">
          <BucketViewOptions />

          <UploadPopover />

          <ButtonGroup>
            <Button onClick={shareFileDialog.trigger}>
              <PlusCircle className="mr-2 h-4 w-4" />
              {t("bucket.header.upload_file")}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button className="!pl-2">
                  <ChevronDownIcon />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  <DropdownMenuItem onClick={newFolderDialog.trigger}>
                    <FolderPlus />
                    {t("file_actions.new_folder")}
                  </DropdownMenuItem>
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </ButtonGroup>

          <FormDialog
            {...shareFileDialog.props}
            title={t("bucket.header.upload_file")}
            description={t("bucket.header.upload_and_share")}
            fields={shareFileFields}
            onSubmit={(data) => startUpload(data.files, bucket.id, folderId)}
            confirmLabel={t("bucket.header.upload")}
          />

          <FormDialog
            {...newFolderDialog.props}
            title={t("file_actions.new_folder_dialog.title")}
            fields={[
              {
                id: "name",
                label: t("file_actions.new_folder_dialog.name_label"),
                type: "text",
                required: true,
              },
            ]}
            onSubmit={(data) => createFolder(data.name)}
            confirmLabel={t("file_actions.new_folder_dialog.create")}
          />
        </div>
      </div>
    </div>
  );
};
