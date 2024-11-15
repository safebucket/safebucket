import React, { FC } from "react";

import { Download, ExternalLink, FolderPlus, Trash2 } from "lucide-react";

import { useFileActions } from "@/components/FileActions/hooks/useFileActions";
import { IFile } from "@/components/bucket-view/helpers/types";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import FormDialog from "@/components/dialogs/components/FormDialog";
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
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface IFileActionsProps {
  children: React.ReactNode;
  file: IFile;
  type: "context" | "dropdown";
}

export const FileActions: FC<IFileActionsProps> = ({
  children,
  file,
  type,
}: IFileActionsProps) => {
  const { createFolder, downloadFile, deleteFile } = useFileActions();
  const newFolderDialog = useDialog();
  const deleteFileDialog = useDialog();

  const Menu = type === "context" ? ContextMenu : DropdownMenu;
  const MenuTrigger =
    type === "context" ? ContextMenuTrigger : DropdownMenuTrigger;
  const MenuContent =
    type === "context" ? ContextMenuContent : DropdownMenuContent;
  const MenuItem = type === "context" ? ContextMenuItem : DropdownMenuItem;

  return (
    <>
      <Menu>
        <MenuTrigger asChild>{children}</MenuTrigger>
        <MenuContent className="w-40">
          <MenuItem onClick={() => downloadFile(file.id, file.name)}>
            <Download className="mr-2 h-4 w-4" />
            Download
          </MenuItem>
          <MenuItem>
            <ExternalLink className="mr-2 h-4 w-4" />
            Share
          </MenuItem>
          <ContextMenuSeparator />
          <MenuItem onClick={newFolderDialog.trigger}>
            <FolderPlus className="mr-2 h-4 w-4" />
            New folder
          </MenuItem>
          <ContextMenuSeparator />
          <MenuItem className="text-red-600" onClick={deleteFileDialog.trigger}>
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </MenuItem>
        </MenuContent>
      </Menu>
      <FormDialog
        {...newFolderDialog.props}
        title="New folder"
        fields={[{ id: "name", label: "Name", type: "text", required: true }]}
        onSubmit={(data) => {
          createFolder(data.name);
        }}
        confirmLabel="Create"
      />
      <CustomAlertDialog
        {...deleteFileDialog.props}
        title={`Delete ${file.name}?`}
        description="Are you sure you want to delete this file? This action cannot be undone."
        confirmLabel="Delete"
        onConfirm={() => deleteFile(file.id, file.name)}
      />
    </>
  );
};
