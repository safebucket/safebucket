import { useCallback, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Upload } from "lucide-react";
import { toast } from "sonner";
import type { FC } from "react";

import type { IPublicShareResponse } from "@/types/share.ts";
import type { IFile } from "@/types/file.ts";
import type { BucketItem } from "@/types/bucket.ts";
import type { ShareUploadHandler } from "@/components/share-view/components/ShareUploadZone.tsx";
import type {
  FileWithPath,
  StagedFile,
} from "@/components/upload/helpers/types.ts";
import { UploadDialog } from "@/components/upload/components/UploadDialog.tsx";
import { toStagedFiles } from "@/components/upload/helpers/file-processing.ts";
import { isFile, isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { ShareHeader } from "@/components/share-view/components/ShareHeader.tsx";
import { ShareContentArea } from "@/components/share-view/components/ShareContentArea.tsx";
import { ShareUploadZone } from "@/components/share-view/components/ShareUploadZone.tsx";
import { Button } from "@/components/ui/button.tsx";
import { FilePreviewDialog } from "@/components/file-actions/components/FilePreviewDialog.tsx";
import {
  getShareDownloadUrl,
  shareContentQueryOptions,
  useShareDownloadMutation,
} from "@/queries/share.ts";
import { downloadFromStorage } from "@/components/file-actions/helpers/api.ts";
import {
  createZipEntries,
  fetchDownloadBlob,
  zipDownload,
} from "@/lib/zip-download.ts";
import { formatFileSize } from "@/lib/utils.ts";

interface IShareContentViewProps {
  shareId: string;
  shareContent: IPublicShareResponse;
}

export const ShareContentView: FC<IShareContentViewProps> = ({
  shareId,
  shareContent,
}) => {
  const { t } = useTranslation();
  const { data: content } = useQuery({
    ...shareContentQueryOptions(shareId),
    initialData: shareContent,
  });
  const [currentFolderId, setCurrentFolderId] = useState<string | undefined>(
    shareContent.folder_id ?? undefined,
  );
  const [folderHistory, setFolderHistory] = useState<Array<string>>([]);
  const [previewItem, setPreviewItem] = useState<IFile | null>(null);
  const [isDownloadingAll, setIsDownloadingAll] = useState(false);
  const uploadFilesRef = useRef<ShareUploadHandler | null>(null);
  const [isUploadDialogOpen, setIsUploadDialogOpen] = useState(false);
  const [stagedUploadFiles, setStagedUploadFiles] = useState<Array<StagedFile>>(
    [],
  );

  const downloadMutation = useShareDownloadMutation(shareId);

  const handleDownload = (file: IFile) => {
    downloadMutation.mutate(
      { fileId: file.id },
      { onSuccess: (data) => downloadFromStorage(data.url, file.name) },
    );
  };

  const zipEntries = useMemo(
    () => createZipEntries(content.files, content.folders),
    [content.files, content.folders],
  );

  const fetchZipFile = useCallback(
    async (file: IFile) => {
      const { url } = await getShareDownloadUrl(shareId, { fileId: file.id });
      return fetchDownloadBlob(url);
    },
    [shareId],
  );

  const handleDownloadAll = async () => {
    setIsDownloadingAll(true);
    try {
      const safeShareName = content.name.replace(/[^a-zA-Z0-9-_]/g, "_");
      const result = await zipDownload({
        entries: zipEntries,
        archiveName: `${safeShareName || "shared-files"}.zip`,
        fetchFile: fetchZipFile,
      });

      if (result.failures.length === result.total) {
        toast.error(t("share_consumer.download_all_failed"));
        return;
      }

      if (result.failures.length > 0) {
        toast.warning(
          t("share_consumer.download_partial", {
            count: result.failures.length,
          }),
        );
      }
    } catch {
      toast.error(t("share_consumer.download_all_failed"));
    } finally {
      setIsDownloadingAll(false);
    }
  };

  const items = useMemo((): Array<BucketItem> => {
    const folderItems = content.folders.filter(
      (folder) =>
        (!currentFolderId && !folder.folder_id) ||
        folder.folder_id === currentFolderId,
    );
    const fileItems = content.files.filter(
      (file) =>
        (!currentFolderId && !file.folder_id) ||
        file.folder_id === currentFolderId,
    );
    return [...folderItems, ...fileItems];
  }, [content, currentFolderId]);

  const handleOpenItem = (item: BucketItem) => {
    if (isFolder(item)) {
      setFolderHistory((previous) => [...previous, currentFolderId ?? ""]);
      setCurrentFolderId(item.id);
    } else if (isFile(item)) {
      setPreviewItem(item);
    }
  };

  const goBack = () => {
    const previousFolderId = folderHistory[folderHistory.length - 1];
    setFolderHistory((history) => history.slice(0, -1));
    setCurrentFolderId(previousFolderId || undefined);
  };

  const handleUploadDialogOpenChange = (open: boolean) => {
    if (!open) {
      setStagedUploadFiles([]);
    }
    setIsUploadDialogOpen(open);
  };

  const handleAddUploadFiles = useCallback((files: Array<FileWithPath>) => {
    setStagedUploadFiles((current) => [...current, ...toStagedFiles(files)]);
  }, []);

  const handleRemoveUploadFile = useCallback((id: string) => {
    setStagedUploadFiles((current) => current.filter((file) => file.id !== id));
  }, []);

  const handleUpload = () => {
    uploadFilesRef.current?.(stagedUploadFiles.map((staged) => staged.file));
    handleUploadDialogOpenChange(false);
  };

  const currentFolderName = currentFolderId
    ? (content.folders.find((folder) => folder.id === currentFolderId)?.name ??
      null)
    : null;
  const totalSize = content.files.reduce((sum, file) => sum + file.size, 0);
  const canUpload =
    content.allow_upload &&
    !(
      content.max_uploads !== null &&
      content.current_uploads >= content.max_uploads
    );
  const contentArea = (
    <ShareContentArea
      items={items}
      folderName={currentFolderName}
      canGoBack={folderHistory.length > 0}
      onGoBack={goBack}
      onOpenItem={handleOpenItem}
      onPreview={setPreviewItem}
      onDownload={handleDownload}
    />
  );

  return (
    <main className="min-h-svh bg-background p-3 sm:p-8 lg:flex lg:items-center">
      <div className="mx-auto grid min-h-[calc(100svh-1.5rem)] w-full max-w-7xl content-start overflow-hidden rounded-2xl border bg-card shadow-xl sm:min-h-[calc(100svh-4rem)] sm:rounded-3xl lg:min-h-[42rem] lg:content-stretch lg:grid-cols-[minmax(20rem,0.8fr)_minmax(0,1.35fr)]">
        <ShareHeader
          shareContent={content}
          totalSize={totalSize}
          isDownloadingAll={isDownloadingAll}
          onDownloadAll={handleDownloadAll}
        />

        <section className="flex min-h-0 min-w-0 flex-col p-5 sm:p-8">
          <header className="flex items-start justify-between gap-4 border-b pb-4">
            <div>
              <h2 className="text-lg font-semibold">
                {t("share_consumer.files")}
              </h2>
              <p className="text-muted-foreground text-sm">
                {t("share_consumer.items_and_size", {
                  count: content.files.length + content.folders.length,
                  size: formatFileSize(totalSize),
                })}
              </p>
            </div>
            {canUpload && (
              <Button
                type="button"
                onClick={() => {
                  setStagedUploadFiles([]);
                  setIsUploadDialogOpen(true);
                }}
              >
                <Upload />
                {t("bucket.header.upload_file")}
              </Button>
            )}
          </header>

          <div className="min-h-0 flex-1 overflow-y-auto pt-4">
            {canUpload ? (
              <ShareUploadZone
                shareId={shareId}
                maxUploadSize={content.max_upload_size}
                folderId={
                  content.type === "bucket"
                    ? currentFolderId
                    : (content.folder_id ?? undefined)
                }
                onReady={(fn) => {
                  uploadFilesRef.current = fn;
                }}
                onFilesDropped={(files) => {
                  setStagedUploadFiles(
                    toStagedFiles(
                      files.map((file) => ({ file, relativePath: "" })),
                    ),
                  );
                  setIsUploadDialogOpen(true);
                }}
              >
                {contentArea}
              </ShareUploadZone>
            ) : (
              contentArea
            )}
          </div>
        </section>
      </div>

      {previewItem && (
        <FilePreviewDialog
          open
          onOpenChange={(isOpen) => {
            if (!isOpen) setPreviewItem(null);
          }}
          file={previewItem}
          fetchUrl={() =>
            downloadMutation.mutateAsync({
              fileId: previewItem.id,
              context: "preview",
            })
          }
          onDownload={() => {
            const fileToDownload = previewItem;
            setPreviewItem(null);
            handleDownload(fileToDownload);
          }}
        />
      )}

      <UploadDialog
        open={isUploadDialogOpen}
        onOpenChange={handleUploadDialogOpenChange}
        stagedFiles={stagedUploadFiles}
        onAddFiles={handleAddUploadFiles}
        onRemoveFile={handleRemoveUploadFile}
        onUpload={handleUpload}
        showExpiration={false}
      />
    </main>
  );
};
