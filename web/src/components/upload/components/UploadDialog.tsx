import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type {
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { UploadAdvancedSection } from "@/components/upload/components/UploadAdvancedSection";
import { UploadDropzone } from "@/components/upload/components/UploadDropzone";
import { UploadFileList } from "@/components/upload/components/UploadFileList";

interface UploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  stagedFiles: Array<StagedFile>;
  onAddFiles: (files: Array<FileWithPath>) => void;
  onRemoveFile: (id: string) => void;
  expiresAt: Date | undefined;
  onExpiresAtChange: (date: Date | undefined) => void;
  isAdvancedOpen: boolean;
  onAdvancedOpenChange: (open: boolean) => void;
  onUpload: () => void;
}

export const UploadDialog: FC<UploadDialogProps> = ({
  open,
  onOpenChange,
  stagedFiles,
  onAddFiles,
  onRemoveFile,
  expiresAt,
  onExpiresAtChange,
  isAdvancedOpen,
  onAdvancedOpenChange,
  onUpload,
}) => {
  const { t } = useTranslation();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-170">
        <DialogHeader>
          <DialogTitle>{t("upload.dialog.title")}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <UploadDropzone onFilesSelected={onAddFiles} />

          <UploadFileList files={stagedFiles} onRemoveFile={onRemoveFile} />

          <UploadAdvancedSection
            isOpen={isAdvancedOpen}
            onOpenChange={onAdvancedOpenChange}
            expiresAt={expiresAt}
            onExpiresAtChange={onExpiresAtChange}
          />
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            disabled={stagedFiles.length === 0}
            onClick={onUpload}
          >
            {t("upload.dialog.upload_button")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
