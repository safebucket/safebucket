import { useCallback, useState } from "react";
import type { FC } from "react";

import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { toStagedFiles } from "@/components/upload/helpers/file-processing";
import type {
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types";

interface ShareUploadDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUploadFiles: (files: Array<File>) => void;
}

export const ShareUploadDialog: FC<ShareUploadDialogProps> = ({
  open,
  onOpenChange,
  onUploadFiles,
}) => {
  const [stagedFiles, setStagedFiles] = useState<Array<StagedFile>>([]);

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setStagedFiles([]);
    }
    onOpenChange(nextOpen);
  };

  const handleAddFiles = useCallback((files: Array<FileWithPath>) => {
    setStagedFiles((current) => [...current, ...toStagedFiles(files)]);
  }, []);

  const handleRemoveFile = useCallback((id: string) => {
    setStagedFiles((current) => current.filter((file) => file.id !== id));
  }, []);

  const handleUpload = () => {
    onUploadFiles(stagedFiles.map((staged) => staged.file));
    handleOpenChange(false);
  };

  return (
    <UploadDialog
      open={open}
      onOpenChange={handleOpenChange}
      stagedFiles={stagedFiles}
      onAddFiles={handleAddFiles}
      onRemoveFile={handleRemoveFile}
      expiresAt={undefined}
      onExpiresAtChange={() => undefined}
      isExpirationEnabled={false}
      onExpirationEnabledChange={() => undefined}
      onUpload={handleUpload}
      showExpiration={false}
    />
  );
};
