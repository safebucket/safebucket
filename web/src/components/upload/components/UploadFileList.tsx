import { Plus, X } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { DragEvent, FC } from "react";
import type {
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import { extractFilesFromDrop } from "@/components/upload/helpers/file-processing";
import { cn, formatFileSize } from "@/lib/utils";

interface UploadFileListProps {
  files: Array<StagedFile>;
  onRemoveFile: (id: string) => void;
  onAddFiles: (files: Array<FileWithPath>) => void;
}

export const UploadFileList: FC<UploadFileListProps> = ({
  files,
  onRemoveFile,
  onAddFiles,
}) => {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);

  const handleDrop = useCallback(
    async (event: DragEvent<HTMLDivElement>) => {
      event.preventDefault();
      event.stopPropagation();
      setIsDragOver(false);

      const droppedFiles = await extractFilesFromDrop(event.dataTransfer);
      if (droppedFiles.length > 0) {
        onAddFiles(droppedFiles);
      }
    },
    [onAddFiles],
  );

  if (files.length === 0) {
    return null;
  }

  const totalSize = files.reduce(
    (total, staged) => total + staged.file.size,
    0,
  );

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between text-xs">
        <p>{t("upload.dialog.file_count", { count: files.length })}</p>
        <p className="text-muted-foreground">
          {t("upload.dialog.file_total", { size: formatFileSize(totalSize) })}
        </p>
      </div>
      <div className="flex max-h-60 flex-col gap-2 overflow-y-auto">
        {files.map((staged) => {
          const size = formatFileSize(staged.file.size);
          const description = staged.relativePath
            ? `${t("upload.dialog.folder_path", { path: staged.relativePath })} · ${size}`
            : size;

          return (
            <Attachment
              key={staged.id}
              size="sm"
              className="min-h-[52px] w-full rounded-xl focus-within:ring-0"
            >
              <AttachmentMedia>
                <FileIconView
                  isFolder={false}
                  extension={staged.extension}
                  className="text-primary"
                />
              </AttachmentMedia>
              <AttachmentContent>
                <AttachmentTitle>{staged.file.name}</AttachmentTitle>
                <AttachmentDescription>{description}</AttachmentDescription>
              </AttachmentContent>
              <AttachmentActions>
                <AttachmentAction
                  onClick={() => onRemoveFile(staged.id)}
                  aria-label={`${t("upload.dialog.remove_file")}: ${staged.file.name}`}
                >
                  <X />
                </AttachmentAction>
              </AttachmentActions>
            </Attachment>
          );
        })}
      </div>
      <div
        className={cn(
          "flex min-h-10 items-center justify-center rounded-xl border-2 border-dashed p-2 transition-colors",
          isDragOver
            ? "border-primary bg-primary/5"
            : "border-muted-foreground/25",
        )}
        onDragOver={(event) => {
          event.preventDefault();
          event.stopPropagation();
          setIsDragOver(true);
        }}
        onDragLeave={(event) => {
          event.preventDefault();
          event.stopPropagation();
          setIsDragOver(false);
        }}
        onDrop={handleDrop}
      >
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => fileInputRef.current?.click()}
        >
          <Plus />
          {t("upload.dialog.add_more_files")}
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(event) => {
            const selectedFiles = event.target.files;
            if (selectedFiles?.length) {
              onAddFiles(
                Array.from(selectedFiles).map((file) => ({
                  file,
                  relativePath: "",
                })),
              );
            }
            event.target.value = "";
          }}
          tabIndex={-1}
        />
      </div>
    </div>
  );
};
