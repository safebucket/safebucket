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
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { RowSelectionState } from "@tanstack/react-table";
import type { BucketItem, IBucket } from "@/types/bucket.ts";
import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { useBulkDownload } from "@/components/bucket-view/hooks/useBulkDownload";
import { GridActionIcon } from "@/components/bucket-view/components/GridActionIcon";
import { FilesTable } from "@/components/bucket-view/components/FilesTable";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useFileActions } from "@/components/file-actions/hooks/useFileActions.ts";
import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { useUploadDialog } from "@/components/upload/hooks/useUploadDialog";
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
import { cn, formatFileSize } from "@/lib/utils";

interface IFilesTabProps {
  bucket: IBucket;
  items: Array<BucketItem>;
  folderId: string | undefined;
  isContributor: boolean;
}

export const FilesTab: FC<IFilesTabProps> = ({
  bucket,
  items,
  folderId,
  isContributor,
}: IFilesTabProps) => {
  const { t } = useTranslation();
  const { createFolder } = useFileActions();
  const uploadDialog = useUploadDialog({ bucketId: bucket.id, folderId });
  const newFolderDialog = useDialog();
  const [query, setQuery] = useState("");
  const [layout, setLayout] = useState<"list" | "grid">("list");
  const [selected, setSelected] = useState<BucketItem | null>(null);
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});

  const clearRowSelection = () => setRowSelection({});

  useEffect(() => {
    setSelected(null);
    setRowSelection({});
  }, [folderId]);

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
                onClick={uploadDialog.open}
              />
              <GridActionIcon
                icon={<FolderPlus className="h-4 w-4" />}
                label={t("file_actions.new_folder")}
                onClick={newFolderDialog.trigger}
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
        <FilesTable
          items={filtered}
          selected={selected}
          setSelected={setSelected}
          rowSelection={rowSelection}
          setRowSelection={setRowSelection}
        />
      ) : (
        <BucketGridView
          items={filtered}
          selected={selected}
          setSelected={setSelected}
        />
      )}

      <FormDialog
        {...newFolderDialog.props}
        title={t("file_actions.new_folder_dialog.title")}
        fields={[
          {
            id: "name",
            label: t("file_actions.new_folder_dialog.name_label"),
            type: "text",
            required: true,
          },
        ]}
        onSubmit={(data) => createFolder(data.name)}
        confirmLabel={t("file_actions.new_folder_dialog.create")}
      />

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

      <UploadDialog
        {...uploadDialog.dialogProps}
        stagedFiles={uploadDialog.stagedFiles}
        onAddFiles={uploadDialog.addFiles}
        onRemoveFile={uploadDialog.removeFile}
        expiresAt={uploadDialog.expiresAt}
        onExpiresAtChange={uploadDialog.setExpiresAt}
        isAdvancedOpen={uploadDialog.isAdvancedOpen}
        onAdvancedOpenChange={uploadDialog.setIsAdvancedOpen}
        onUpload={uploadDialog.handleUpload}
      />
    </div>
  );
};
