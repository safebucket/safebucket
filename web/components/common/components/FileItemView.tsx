import React, { FC } from "react";

import { cn } from "@/lib/utils";

import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { IFile } from "@/components/bucket-view/helpers/types";
import { Card } from "@/components/ui/card";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";

interface IFileViewProps {
  file: IFile;
  selected: IFile | null;
  setSelected: (file: IFile) => void;
  onDoubleClick: (file: IFile) => void;
}

export const FileItemView: FC<IFileViewProps> = ({
  file,
  selected,
  setSelected,
  onDoubleClick,
}: IFileViewProps) => {
  return (
    <div className={cn("space-y-3")}>
      <ContextMenu>
        <ContextMenuTrigger>
          <Card
            key={file.id}
            className={`flex cursor-pointer flex-col gap-4 p-4 ${selected?.id === file.id ? "bg-primary text-primary-foreground" : ""}`}
            onClick={() => setSelected(file)}
            onDoubleClick={() => onDoubleClick(file)}
          >
            <div className="flex items-center gap-4">
              <div
                className={`flex aspect-square w-12 items-center justify-center rounded-md bg-muted ${
                  selected?.id === file.id
                    ? "bg-primary-foreground text-primary"
                    : ""
                }`}
              >
                <FileIconView extension={file.type} className="h-6 w-6" />
              </div>
              <div className="flex-1">
                <h3
                  className={`truncate font-medium ${selected?.id === file.id ? "text-primary-foreground" : ""}`}
                >
                  {file.name}
                </h3>
                <p
                  className={`text-sm ${selected?.id === file.id ? "text-primary-foreground" : "text-muted-foreground"}`}
                >
                  {file.size}
                </p>
              </div>
            </div>
            <div
              className={`text-sm ${selected?.id === file.id ? "text-primary-foreground" : "text-muted-foreground"}`}
            >
              Modified: {file.modified}
            </div>
          </Card>
        </ContextMenuTrigger>
        <ContextMenuContent className="w-40">
          <ContextMenuItem>Download</ContextMenuItem>
          <ContextMenuItem>Delete</ContextMenuItem>
          <ContextMenuSeparator />
          <ContextMenuItem>Share</ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
    </div>
  );
};
