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
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { FileWithPath } from "@/components/upload/helpers/types";
import type { BucketItem, IBucket } from "@/types/bucket.ts";
import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { useBucketViewContext } from "@/components/bucket-view/hooks/useBucketViewContext";
import { useBulkDownload } from "@/components/bucket-view/hooks/useBulkDownload";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { FilesTable } from "@/components/bucket-view/components/FilesTable";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DragDropZone } from "@/components/upload/components/DragDropZone";
import { cn, formatFileSize } from "@/lib/utils";

interface IFilesTabProps {
  bucket: IBucket;
  items: Array<BucketItem>;
  onFilesDropped: (files: Array<FileWithPath>) => void;
  onUpload: () => void;
  onNewFolder: () => void;
  isContributor: boolean;
}

export const FilesTab: FC<IFilesTabProps> = ({
  bucket,
  items,
  onFilesDropped,
  onUpload,
  onNewFolder,
  isContributor,
}: IFilesTabProps) => {
  const { t } = useTranslation();
  const { rowSelection, clearRowSelection } = useBucketViewContext();
  const [query, setQuery] = useState("");
  const [layout, setLayout] = useState<"list" | "grid">("list");

  const {
    start,
    dismissBlocked,
    blocked,
    isRunning,
    fileCount,
    maxBytes,
    maxFiles,
  } = useBulkDownload({ bucket, rowSelection, clearRowSelection });

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((item) => item.name.toLowerCase().includes(q));
  }, [items, query]);

  const selectedCount = Object.values(rowSelection).filter(Boolean).length;
  const hasSelection = selectedCount > 0;

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <div className="relative w-full max-w-xs">
          <Search className="text-muted-foreground absolute top-1/2 left-2.5 h-4 w-4 -translate-y-1/2" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
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
            onClick={start}
            disabled={!hasSelection || isRunning || fileCount === 0}
          />
          {isContributor ? (
            <GridActionIcon
              icon={<Trash2 className="h-4 w-4" />}
              label={t("bucket.view.actions.bulk_trash")}
              onClick={clearRowSelection}
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
              onClick={() => setLayout("list")}
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
              onClick={() => setLayout("grid")}
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      {layout === "list" ? (
        <DragDropZone bucketId={bucket.id} onFilesDropped={onFilesDropped}>
          <FilesTable items={filtered} />
        </DragDropZone>
      ) : (
        <BucketGridView
          items={filtered}
          bucketId={bucket.id}
          onFilesDropped={onFilesDropped}
        />
      )}

      <AlertDialog
        open={blocked !== null}
        onOpenChange={(isOpen) => {
          if (!isOpen) dismissBlocked();
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("bucket.bulk_download.over_limit_title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("bucket.bulk_download.over_limit_body", {
                count: blocked?.count ?? 0,
                size: formatFileSize(blocked?.bytes ?? 0),
                maxCount: maxFiles,
                maxSize: formatFileSize(maxBytes),
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={dismissBlocked}>
              {t("common.ok")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};
