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
  onUpload: () => void;
  showExpiration?: boolean;
  expiresAt?: Date | undefined;
  onExpiresAtChange?: (date: Date | undefined) => void;
  isExpirationEnabled?: boolean;
  onExpirationEnabledChange?: (enabled: boolean) => void;
}

export const UploadDialog: FC<UploadDialogProps> = ({
  open,
  onOpenChange,
  stagedFiles,
  onAddFiles,
  onRemoveFile,
  onUpload,
  showExpiration = true,
  expiresAt,
  onExpiresAtChange = () => undefined,
  isExpirationEnabled = false,
  onExpirationEnabledChange = () => undefined,
}) => {
  const { t } = useTranslation();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-4 rounded-2xl p-5 sm:max-w-[36rem]">
        <DialogHeader>
          <DialogTitle>{t("upload.dialog.title")}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {stagedFiles.length === 0 && (
            <UploadDropzone onFilesSelected={onAddFiles} />
          )}

          <UploadFileList
            files={stagedFiles}
            onRemoveFile={onRemoveFile}
            onAddFiles={onAddFiles}
          />

          {showExpiration && (
            <UploadAdvancedSection
              isEnabled={isExpirationEnabled}
              onEnabledChange={onExpirationEnabledChange}
              expiresAt={expiresAt}
              onExpiresAtChange={onExpiresAtChange}
            />
          )}
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
