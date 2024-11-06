import React, { FC } from "react";

import { useFileActions } from "@/components/FileActions/hooks/useFileActions";
import { IFile } from "@/components/bucket-view/helpers/types";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
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
  const { deleteFile } = useFileActions();

  const Menu = type === "context" ? ContextMenu : DropdownMenu;
  const MenuTrigger =
    type === "context" ? ContextMenuTrigger : DropdownMenuTrigger;
  const MenuContent =
    type === "context" ? ContextMenuContent : DropdownMenuContent;
  const MenuItem = type === "context" ? ContextMenuItem : DropdownMenuItem;

  return (
    <AlertDialog>
      <Menu>
        <MenuTrigger asChild>{children}</MenuTrigger>
        <MenuContent className="w-40">
          <MenuItem>Download</MenuItem>
          <MenuItem>Share</MenuItem>
          <ContextMenuSeparator />
          <AlertDialogTrigger asChild>
            <MenuItem className="text-red-600">Delete</MenuItem>
          </AlertDialogTrigger>
        </MenuContent>
      </Menu>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {file.name}?</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete this file? This action cannot be
            undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={() => deleteFile(file.id, file.name)}>
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
};
