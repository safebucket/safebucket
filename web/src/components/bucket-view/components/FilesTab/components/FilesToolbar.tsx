import {
  Download,
  FolderPlus,
  LayoutGrid,
  List,
  Loader2,
  Plus,
  Search,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

interface IFilesToolbarProps {
  query: string;
  onQueryChange: (value: string) => void;
  layout: "list" | "grid";
  onLayoutChange: (layout: "list" | "grid") => void;
  isContributor: boolean;
  hasSelection: boolean;
  isRunning: boolean;
  fileCount: number;
  onUpload: () => void;
  onNewFolder: () => void;
  onBulkDownload: () => void;
  onBulkTrash: () => void;
}

export const FilesToolbar: FC<IFilesToolbarProps> = ({
  query,
  onQueryChange,
  layout,
  onLayoutChange,
  isContributor,
  hasSelection,
  isRunning,
  fileCount,
  onUpload,
  onNewFolder,
  onBulkDownload,
  onBulkTrash,
}) => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center justify-between gap-2.5">
      <div className="relative w-full max-w-xs">
        <Search className="text-muted-foreground absolute top-1/2 left-2.5 h-4 w-4 -translate-y-1/2" />
        <Input
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder={t("bucket.view.search_placeholder")}
          className="h-8 pl-8"
        />
      </div>
      <div className="flex items-center gap-1">
        {isContributor ? (
          <>
            <GridActionIcon
              icon={<Plus className="h-4 w-4" />}
              label={t("bucket.header.upload_file")}
              onClick={onUpload}
            />
            <GridActionIcon
              icon={<FolderPlus className="h-4 w-4" />}
              label={t("file_actions.new_folder")}
              onClick={onNewFolder}
            />
          </>
        ) : null}
        <GridActionIcon
          icon={
            isRunning ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Download className="h-4 w-4" />
            )
          }
          label={t("bucket.view.actions.bulk_download")}
          onClick={onBulkDownload}
          disabled={!hasSelection || isRunning || fileCount === 0}
        />
        {isContributor ? (
          <GridActionIcon
            icon={<Trash2 className="h-4 w-4" />}
            label={t("bucket.view.actions.bulk_trash")}
            onClick={onBulkTrash}
            disabled={!hasSelection || isRunning}
            destructive
          />
        ) : null}

        <div className="border-border ml-1 flex items-center rounded-lg border p-0.5">
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "h-7 w-7",
              layout === "list" && "bg-accent text-accent-foreground",
            )}
            aria-label={t("bucket.header.list_view")}
            onClick={() => onLayoutChange("list")}
          >
            <List className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "h-7 w-7",
              layout === "grid" && "bg-accent text-accent-foreground",
            )}
            aria-label={t("bucket.header.grid_view")}
            onClick={() => onLayoutChange("grid")}
          >
            <LayoutGrid className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
};
