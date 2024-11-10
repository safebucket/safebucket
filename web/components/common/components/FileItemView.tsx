import React, { FC } from "react";

import { cn, formatDate, formatFileSize } from "@/lib/utils";

import { FileActions } from "@/components/FileActions/FileActions";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { FileType, IFile } from "@/components/bucket-view/helpers/types";
import { Card } from "@/components/ui/card";

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
  const isSelected = selected?.id === file.id;

  return (
    <div className="space-y-3">
      <FileActions file={file} type="context">
        <Card
          key={file.id}
          className={cn(
            "flex cursor-pointer flex-col gap-4 p-4",
            isSelected && "bg-primary text-primary-foreground",
          )}
          onClick={() => setSelected(file)}
          onDoubleClick={() => onDoubleClick(file)}
        >
          <div className="flex items-center gap-4">
            <div
              className={cn(
                "flex aspect-square w-12 items-center justify-center rounded-md bg-muted",
                isSelected && "bg-primary-foreground text-primary",
              )}
            >
              <FileIconView
                className="h-6 w-6"
                type={file.type}
                extension={file.extension}
              />
            </div>
            <div className="flex-1">
              <h3
                className={cn(
                  "truncate font-medium",
                  isSelected && "text-primary-foreground",
                )}
              >
                {file.name}
              </h3>
              <p
                className={cn(
                  "text-sm",
                  isSelected
                    ? "text-primary-foreground"
                    : "text-muted-foreground",
                )}
              >
                {file.type === FileType.folder ? "-" : formatFileSize(file.size)}
              </p>
            </div>
          </div>
          <div
            className={cn(
              "text-sm",
              isSelected ? "text-primary-foreground" : "text-muted-foreground",
            )}
          >
            Uploaded: {formatDate(file.created_at)}
          </div>
        </Card>
      </FileActions>
    </div>
  );
};
