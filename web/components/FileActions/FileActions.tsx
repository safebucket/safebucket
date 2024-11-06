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

interface IFileActionsProps {
  children: React.ReactNode;
  file: IFile;
}

export const FileActions: FC<IFileActionsProps> = ({
  children,
  file,
}: IFileActionsProps) => {
  const { deleteFile } = useFileActions();

  return (
    <AlertDialog>
      <ContextMenu>
        <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
        <ContextMenuContent className="w-40">
          <ContextMenuItem>Download</ContextMenuItem>
          <ContextMenuItem>Share</ContextMenuItem>
          <ContextMenuSeparator />
          <AlertDialogTrigger asChild>
            <ContextMenuItem className="text-red-600">Delete</ContextMenuItem>
          </AlertDialogTrigger>
        </ContextMenuContent>
      </ContextMenu>
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
