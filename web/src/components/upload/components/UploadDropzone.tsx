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
      // Reset input so the same file can be re-selected
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [onFilesSelected],
  );

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed p-8 transition-colors",
        isDragOver
          ? "border-primary bg-primary/5"
          : "border-muted-foreground/25 hover:border-muted-foreground/50",
      )}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <Upload className="text-muted-foreground h-10 w-10" />
      <p className="text-muted-foreground text-sm">
        {t("upload.dialog.dropzone_text")}
      </p>
      <Button type="button" variant="default" onClick={handleBrowseClick}>
        {t("upload.dialog.browse")}
      </Button>
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
