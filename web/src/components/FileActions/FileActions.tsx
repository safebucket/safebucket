import { useTranslation } from "react-i18next";

import {
  ArchiveRestore,
  Download,
  ExternalLink,
  FolderPlus,
  Trash2,
} from "lucide-react";
import type { FC, ReactNode } from "react";

import type { IFile } from "@/types/file.ts";
import { FileStatus } from "@/types/file.ts";
import { useFileActions } from "@/components/FileActions/hooks/useFileActions";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface IFileActionsProps {
  children: ReactNode;
  file: IFile;
  type: "context" | "dropdown";
  trashMode?: boolean;
  onRestore?: (fileId: string, fileName: string) => void;
  onPermanentDelete?: (fileId: string, fileName: string) => void;
}

export const FileActions: FC<IFileActionsProps> = ({
  children,
  file,
  type,
  trashMode = false,
  onRestore,
  onPermanentDelete,
}: IFileActionsProps) => {
  const { t } = useTranslation();
  const { createFolder, downloadFile, deleteFile } = useFileActions();
  const newFolderDialog = useDialog();
  const deleteFileDialog = useDialog();

  const Menu = type === "context" ? ContextMenu : DropdownMenu;
  const MenuTrigger =
    type === "context" ? ContextMenuTrigger : DropdownMenuTrigger;
  const MenuContent =
    type === "context" ? ContextMenuContent : DropdownMenuContent;
  const MenuItem = type === "context" ? ContextMenuItem : DropdownMenuItem;
  const Separator =
    type === "context" ? ContextMenuSeparator : DropdownMenuSeparator;

  const isFileTrashed = file.status === FileStatus.trashed;

  return (
    <>
      <Menu>
        <MenuTrigger asChild>{children}</MenuTrigger>
        <MenuContent className={trashMode ? "w-[180px]" : "w-40"}>
          {trashMode ? (
            <>
              <MenuItem onClick={() => onRestore?.(file.id, file.name)}>
                <ArchiveRestore className="mr-2 h-4 w-4" />
                {t("bucket.trash_view.restore")}
              </MenuItem>
              <Separator />
              <MenuItem
                className="text-red-600"
                onClick={() => onPermanentDelete?.(file.id, file.name)}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                {t("bucket.trash_view.delete_permanently")}
              </MenuItem>
            </>
          ) : (
            <>
              <MenuItem
                onClick={() =>
                  !isFileTrashed && downloadFile(file.id, file.name)
                }
                disabled={isFileTrashed}
              >
                <Download className="mr-2 h-4 w-4" />
                {t("file_actions.download")}
              </MenuItem>
              <MenuItem>
                <ExternalLink className="mr-2 h-4 w-4" />
                {t("file_actions.share")}
              </MenuItem>
              <Separator />
              <MenuItem onClick={newFolderDialog.trigger}>
                <FolderPlus className="mr-2 h-4 w-4" />
                {t("file_actions.new_folder")}
              </MenuItem>
              <Separator />
              <MenuItem
                className="text-red-600"
                onClick={deleteFileDialog.trigger}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                {t("file_actions.move_to_trash")}
              </MenuItem>
            </>
          )}
        </MenuContent>
      </Menu>
      {!trashMode && (
        <>
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
          <CustomAlertDialog
            {...deleteFileDialog.props}
            title={t("file_actions.delete_dialog.title", {
              fileName: file.name,
            })}
            description={t("file_actions.delete_dialog.description")}
            confirmLabel={t("file_actions.delete_dialog.confirm")}
            onConfirm={() => deleteFile(file.id, file.name)}
          />
        </>
      )}
    </>
  );
};
