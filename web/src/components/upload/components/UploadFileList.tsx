import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { StagedFile } from "@/components/upload/helpers/types";
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
      <div className="flex max-h-60 flex-col gap-2 overflow-y-auto">
        {files.map((staged) => {
          const size = formatFileSize(staged.file.size);
          const description = staged.relativePath
            ? `${t("upload.dialog.folder_path", { path: staged.relativePath })} · ${size}`
            : size;

          return (
            <Attachment key={staged.id} size="sm" className="w-full">
              <AttachmentMedia>
                <FileIconView isFolder={false} extension={staged.extension} />
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
    </div>
  );
};
