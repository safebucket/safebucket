import { Upload } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { DragEvent, FC } from "react";
import type { FileWithPath } from "@/components/upload/helpers/types";
import { Button } from "@/components/ui/button";
import { extractFilesFromDrop } from "@/components/upload/helpers/file-processing";
import { cn } from "@/lib/utils";

interface UploadDropzoneProps {
  onFilesSelected: (files: Array<FileWithPath>) => void;
}

export const UploadDropzone: FC<UploadDropzoneProps> = ({
  onFilesSelected,
}) => {
  const { t } = useTranslation();
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(true);
  }, []);

  const handleDragLeave = useCallback((e: DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragOver(false);
  }, []);

  const handleDrop = useCallback(
    async (e: DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      e.stopPropagation();
      setIsDragOver(false);

      const files = await extractFilesFromDrop(e.dataTransfer);
      if (files.length > 0) {
        onFilesSelected(files);
      }
    },
    [onFilesSelected],
  );

  const handleBrowseClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleFileInputChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const selectedFiles = e.target.files;
      if (selectedFiles && selectedFiles.length > 0) {
        const files: Array<FileWithPath> = Array.from(selectedFiles).map(
          (file) => ({
            file,
            relativePath: "",
          }),
        );
        onFilesSelected(files);
      }
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [onFilesSelected],
  );

  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-xl border-2 border-dashed transition-colors",
        isDragOver
          ? "border-primary bg-primary/5"
          : "border-muted-foreground/25 hover:border-muted-foreground/50",
        "min-h-54 flex-col gap-2 p-8",
      )}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div className="bg-primary/10 flex size-12 items-center justify-center rounded-full">
        <Upload className="text-primary size-5" />
      </div>
      {isDragOver ? (
        <div className="text-center">
          <p className="text-primary text-sm font-medium">
            {t("upload.dialog.drop_to_upload")}
          </p>
        </div>
      ) : (
        <div className="flex flex-col items-center gap-2">
          <p className="text-sm font-medium">
            {t("upload.dialog.dropzone_text")}
          </p>
          <span className="text-muted-foreground text-xs">
            {t("upload.dialog.or")}
          </span>
          <Button type="button" size="sm" onClick={handleBrowseClick}>
            {t("upload.dialog.browse")}
          </Button>
        </div>
      )}
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={handleFileInputChange}
        tabIndex={-1}
      />
    </div>
  );
};
