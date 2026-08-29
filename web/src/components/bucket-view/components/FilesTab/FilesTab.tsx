import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import type { FC } from "react";
import type { RowSelectionState } from "@tanstack/react-table";
import type { BucketItem, IBucket } from "@/types/bucket.ts";
import { BucketGridView } from "@/components/bucket-view/components/BucketGridView";
import { FolderPathBar } from "@/components/bucket-view/components/FolderPathBar";
import { useBulkDownload } from "@/components/bucket-view/hooks/useBulkDownload";
import { FilesTable } from "@/components/bucket-view/components/FilesTable";
import { FilesDndProvider } from "@/components/bucket-view/components/FilesDndProvider/FilesDndProvider";
import { FilesToolbar } from "@/components/bucket-view/components/FilesTab/components/FilesToolbar";
import { CustomAlertDialog } from "@/components/dialogs/components/CustomAlertDialog";
import { FormDialog } from "@/components/dialogs/components/FormDialog";
import { useDialog } from "@/components/dialogs/hooks/useDialog";
import { useFileActions } from "@/components/file-actions/hooks/useFileActions.ts";
import { UploadDialog } from "@/components/upload/components/UploadDialog";
import { useUploadDialog } from "@/components/upload/hooks/useUploadDialog";
import { formatFileSize } from "@/lib/utils";

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

  const clearRowSelection = useCallback(() => setRowSelection({}), []);

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

  const selectedIds = useMemo(
    () =>
      Object.entries(rowSelection)
        .filter(([, v]) => v)
        .map(([id]) => id),
    [rowSelection],
  );

  return (
    <FilesDndProvider
      bucketId={bucket.id}
      items={items}
      selectedIds={selectedIds}
      onMoveSuccess={clearRowSelection}
    >
      <div className="flex flex-col gap-4">
        <FolderPathBar bucket={bucket} folderId={folderId} />

        <FilesToolbar
          query={query}
          onQueryChange={setQuery}
          layout={layout}
          onLayoutChange={setLayout}
          isContributor={isContributor}
          hasSelection={hasSelection}
          isRunning={isRunning}
          fileCount={fileCount}
          onUpload={uploadDialog.open}
          onNewFolder={newFolderDialog.trigger}
          onBulkDownload={start}
          onBulkTrash={clearRowSelection}
        />

        {layout === "list" ? (
          <FilesTable
            items={filtered}
            selected={selected}
            setSelected={setSelected}
            rowSelection={rowSelection}
            setRowSelection={setRowSelection}
            enabled={isContributor}
            selectedIds={selectedIds}
          />
        ) : (
          <BucketGridView
            items={filtered}
            selected={selected}
            setSelected={setSelected}
            enabled={isContributor}
            selectedIds={selectedIds}
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

        <CustomAlertDialog
          open={blocked !== null}
          onOpenChange={(isOpen) => {
            if (!isOpen) dismissBlocked();
          }}
          title={t("bucket.bulk_download.over_limit_title")}
          description={t("bucket.bulk_download.over_limit_body", {
            count: blocked?.count ?? 0,
            size: formatFileSize(blocked?.bytes ?? 0),
            maxCount: maxFiles,
            maxSize: formatFileSize(maxBytes),
          })}
          confirmLabel={t("common.ok")}
          showCancel={false}
          onConfirm={dismissBlocked}
        />

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
    </FilesDndProvider>
  );
};
