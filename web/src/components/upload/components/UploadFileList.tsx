import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { StagedFile } from "@/components/upload/helpers/types";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatFileSize } from "@/lib/utils";

interface UploadFileListProps {
  files: Array<StagedFile>;
  onRemoveFile: (id: string) => void;
}

export const UploadFileList: FC<UploadFileListProps> = ({
  files,
  onRemoveFile,
}) => {
  const { t } = useTranslation();

  if (files.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1">
      <p className="text-muted-foreground text-xs">
        {t("upload.dialog.file_count", { count: files.length })}
      </p>
      <div className="max-h-60 space-y-1 overflow-y-auto">
        {files.map((staged) => (
          <div
            key={staged.id}
            className="bg-muted/50 flex items-center gap-3 rounded-md px-3 py-2"
          >
            <FileIconView
              className="text-primary h-5 w-5 shrink-0"
              isFolder={false}
              extension={staged.extension}
            />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{staged.file.name}</p>
              {staged.relativePath && (
                <p className="text-muted-foreground truncate text-xs">
                  {t("upload.dialog.folder_path", {
                    path: staged.relativePath,
                  })}
                </p>
              )}
            </div>
            {staged.extension && (
              <Badge variant="secondary" className="shrink-0 text-xs">
                {staged.extension.toUpperCase()}
              </Badge>
            )}
            <span className="text-muted-foreground shrink-0 text-xs">
              {formatFileSize(staged.file.size)}
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="h-6 w-6 shrink-0"
              onClick={() => onRemoveFile(staged.id)}
              aria-label={`${t("upload.dialog.remove_file")}: ${staged.file.name}`}
            >
              <X className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
};
