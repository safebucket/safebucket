import { useTranslation } from "react-i18next";

import { FileText, FolderOpen, HardDrive } from "lucide-react";
import type { FC } from "react";

import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import type { ShareScope } from "@/components/quick-share/QuickShareDialog";
import { QuickShareFileList } from "@/components/quick-share/components/QuickShareFileList";
import { QuickShareFolderList } from "@/components/quick-share/components/QuickShareFolderList";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

interface IQuickShareScopeStepProps {
  scope: ShareScope;
  selectedFileIds: Array<string>;
  selectedFolderId: string | null;
  files: Array<IFile>;
  folders: Array<IFolder>;
  onScopeChange: (scope: ShareScope) => void;
  onToggleFile: (fileId: string) => void;
  onSelectFolder: (folderId: string) => void;
}

export const QuickShareScopeStep: FC<IQuickShareScopeStepProps> = ({
  scope,
  selectedFileIds,
  selectedFolderId,
  files,
  folders,
  onScopeChange,
  onToggleFile,
  onSelectFolder,
}) => {
  const { t } = useTranslation();

  const scopeOptions = [
    {
      value: "files" as ShareScope,
      icon: FileText,
      label: t("quick_share.scope_files"),
      description: t("quick_share.scope_files_description"),
    },
    {
      value: "folder" as ShareScope,
      icon: FolderOpen,
      label: t("quick_share.scope_folder"),
      description: t("quick_share.scope_folder_description"),
    },
    {
      value: "bucket" as ShareScope,
      icon: HardDrive,
      label: t("quick_share.scope_bucket"),
      description: t("quick_share.scope_bucket_description"),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-3">
        {scopeOptions.map(({ value, icon: Icon, label, description }) => (
          <button
            key={value}
            type="button"
            onClick={() => onScopeChange(value)}
            className={cn(
              "flex flex-col items-center gap-2 rounded-lg border-2 p-4 text-center transition-all",
              scope === value
                ? "border-primary bg-primary/5 ring-primary/20 ring-1"
                : "hover:bg-accent border-transparent hover:border-border",
            )}
          >
            <div
              className={cn(
                "flex h-10 w-10 items-center justify-center rounded-full",
                scope === value
                  ? "bg-primary/10 text-primary"
                  : "bg-muted text-muted-foreground",
              )}
            >
              <Icon className="h-5 w-5" />
            </div>
            <span className="text-sm font-medium">{label}</span>
            <span className="text-muted-foreground text-xs">{description}</span>
          </button>
        ))}
      </div>

      {scope === "files" && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <Label>{t("quick_share.scope_files")}</Label>
            {selectedFileIds.length > 0 && (
              <span className="text-muted-foreground text-xs">
                {t("quick_share.selected_count", {
                  count: selectedFileIds.length,
                })}
              </span>
            )}
          </div>
          <QuickShareFileList
            files={files}
            selectedIds={selectedFileIds}
            onToggle={onToggleFile}
          />
        </div>
      )}

      {scope === "folder" && (
        <div className="space-y-2">
          <Label>{t("quick_share.scope_folder")}</Label>
          <QuickShareFolderList
            folders={folders}
            selectedId={selectedFolderId}
            onSelect={onSelectFolder}
          />
        </div>
      )}
    </div>
  );
};
