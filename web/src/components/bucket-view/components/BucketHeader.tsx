import { ChevronDownIcon, FolderPlus, PlusCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { IBucket } from "@/types/bucket.ts";
import { BucketViewOptions } from "@/components/bucket-view/components/BucketViewOptions";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useFileActions } from "@/components/FileActions/hooks/useFileActions.ts";
import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu.tsx";
import { useBucketPermissions } from "@/hooks/usePermissions";

interface IBucketHeaderProps {
  bucket: IBucket;
  onOpenUploadDialog: () => void;
}

export const BucketHeader: FC<IBucketHeaderProps> = ({
  bucket,
  onOpenUploadDialog,
}: IBucketHeaderProps) => {
  const { t } = useTranslation();
  const { isContributor } = useBucketPermissions(bucket.id);
  const newFolderDialog = useDialog();

  const { createFolder } = useFileActions();

  return (
    <div className="flex-1">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{bucket.name}</h1>
        <div className="flex items-center gap-4">
          <BucketViewOptions />

          {isContributor ? (
            <>
              <ButtonGroup>
                <Button onClick={onOpenUploadDialog}>
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
            </>
          ) : (
            <div className="text-sm text-muted-foreground">
              {t("bucket.header.view_only")}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
