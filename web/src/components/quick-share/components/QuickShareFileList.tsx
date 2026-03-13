import { useTranslation } from "react-i18next";

import { Check } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file.ts";
import { FileIconView } from "@/components/bucket-view/components/FileIconView";
import { cn } from "@/lib/utils";

interface IQuickShareFileListProps {
  files: Array<IFile>;
  selectedIds: Array<string>;
  onToggle: (fileId: string) => void;
}

export const QuickShareFileList: FC<IQuickShareFileListProps> = ({
  files,
  selectedIds,
  onToggle,
}) => {
  const { t } = useTranslation();

  if (files.length === 0) {
    return (
      <div className="text-muted-foreground flex items-center justify-center rounded-lg border border-dashed py-8 text-sm">
        {t("quick_share.no_files")}
      </div>
    );
  }

  return (
    <div className="max-h-60 space-y-1 overflow-y-auto">
      {files.map((file) => {
        const selected = selectedIds.includes(file.id);
        return (
          <button
            key={file.id}
            type="button"
            onClick={() => onToggle(file.id)}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left text-sm transition-all",
              selected
                ? "border-primary bg-primary/5 ring-primary/20 ring-1"
                : "hover:bg-accent border-transparent",
            )}
          >
            <div
              className={cn(
                "flex h-5 w-5 shrink-0 items-center justify-center rounded border transition-colors",
                selected
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-muted-foreground/30",
              )}
            >
              {selected && <Check className="h-3 w-3" />}
            </div>
            <FileIconView
              className="text-muted-foreground h-4 w-4 shrink-0"
              isFolder={false}
              extension={file.extension}
            />
            <span className="flex-1 truncate">{file.name}</span>
          </button>
        );
      })}
    </div>
  );
};
