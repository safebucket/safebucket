import { useTranslation } from "react-i18next";

import { FolderOpen } from "lucide-react";
import type { FC } from "react";

import type { IFolder } from "@/types/folder.ts";
import { cn } from "@/lib/utils";

interface IQuickShareFolderListProps {
  folders: Array<IFolder>;
  selectedId: string | null;
  onSelect: (folderId: string) => void;
}

export const QuickShareFolderList: FC<IQuickShareFolderListProps> = ({
  folders,
  selectedId,
  onSelect,
}) => {
  const { t } = useTranslation();

  if (folders.length === 0) {
    return (
      <div className="text-muted-foreground flex items-center justify-center rounded-lg border border-dashed py-8 text-sm">
        {t("quick_share.no_folders")}
      </div>
    );
  }

  return (
    <div className="max-h-60 space-y-1 overflow-y-auto">
      {folders.map((folder) => {
        const selected = folder.id === selectedId;
        return (
          <button
            key={folder.id}
            type="button"
            onClick={() => onSelect(folder.id)}
            className={cn(
              "flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left text-sm transition-all",
              selected
                ? "border-primary bg-primary/5 ring-primary/20 ring-1"
                : "hover:bg-accent border-transparent",
            )}
          >
            <div
              className={cn(
                "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border-2 transition-colors",
                selected ? "border-primary" : "border-muted-foreground/30",
              )}
            >
              {selected && (
                <div className="bg-primary h-2.5 w-2.5 rounded-full" />
              )}
            </div>
            <FolderOpen className="text-muted-foreground h-4 w-4 shrink-0" />
            <span className="flex-1 truncate">{folder.name}</span>
          </button>
        );
      })}
    </div>
  );
};
