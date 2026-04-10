import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { VisibilityState } from "@tanstack/react-table";
import type { FC } from "react";

import type { IPublicShareResponse } from "@/types/share.ts";
import type { IFile } from "@/types/file.ts";
import type { BucketItem } from "@/types/bucket.ts";
import type { ShareUploadHandler } from "@/components/share-view/components/ShareUploadZone.tsx";
import { isFolder } from "@/components/bucket-view/helpers/utils.ts";
import { createColumns } from "@/components/share-view/components/ShareColumns.tsx";
import { ShareHeader } from "@/components/share-view/components/ShareHeader.tsx";
import { ShareContentArea } from "@/components/share-view/components/ShareContentArea.tsx";
import { ShareUploadZone } from "@/components/share-view/components/ShareUploadZone.tsx";
import { useIsMobile } from "@/components/ui/hooks/use-mobile.tsx";
import { useShareDownloadMutation } from "@/queries/share.ts";
import { downloadFromStorage } from "@/components/file-actions/helpers/api.ts";
import { errorToast } from "@/components/ui/hooks/use-toast.ts";

export type ViewMode = "list" | "grid";

interface IShareContentViewProps {
  shareId: string;
  shareContent: IPublicShareResponse;
  token: string | null;
}

export const ShareContentView: FC<IShareContentViewProps> = ({
  shareId,
  shareContent,
  token,
}) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const [currentFolderId, setCurrentFolderId] = useState<string | undefined>(
    shareContent.type === "folder" ? shareContent.id : undefined,
  );
  const [folderHistory, setFolderHistory] = useState<Array<string>>([]);
  const uploadFilesRef = useRef<ShareUploadHandler | null>(null);

  const downloadMutation = useShareDownloadMutation(shareId, token);

  const handleDownload = (file: IFile) => {
    downloadMutation.mutate(file.id, {
      onSuccess: (data) => {
        downloadFromStorage(data.url, file.name);
      },
      onError: (error: Error) => errorToast(error),
    });
  };

  const columns = useMemo(() => createColumns(t, handleDownload), [t, token]);

  const columnVisibility = useMemo(
    (): VisibilityState =>
      isMobile ? { size: false, type: false, created_at: false } : {},
    [isMobile],
  );

  const items = useMemo((): Array<BucketItem> => {
    const folderItems = shareContent.folders.filter(
      (folder) =>
        (!currentFolderId && !folder.folder_id) ||
        folder.folder_id === currentFolderId,
    );
    const fileItems = shareContent.files.filter(
      (file) =>
        (!currentFolderId && !file.folder_id) ||
        file.folder_id === currentFolderId,
    );
    return [...folderItems, ...fileItems];
  }, [shareContent, currentFolderId]);

  const openFolder = (item: BucketItem) => {
    if (isFolder(item)) {
      setFolderHistory((prev) => [...prev, currentFolderId ?? ""]);
      setCurrentFolderId(item.id);
    }
  };

  const goBack = () => {
    const prev = folderHistory[folderHistory.length - 1];
    setFolderHistory((h) => h.slice(0, -1));
    setCurrentFolderId(prev || undefined);
  };

  const currentFolderName = currentFolderId
    ? (shareContent.folders.find((f) => f.id === currentFolderId)?.name ?? null)
    : null;

  const handleUploadFiles = (files: Array<File>) => {
    uploadFilesRef.current?.(files);
  };

  const contentArea = (
    <ShareContentArea
      items={items}
      columns={columns}
      columnVisibility={columnVisibility}
      viewMode={viewMode}
      folderName={currentFolderName}
      canGoBack={folderHistory.length > 0}
      onGoBack={goBack}
      onOpenFolder={openFolder}
      onDownload={handleDownload}
    />
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ShareHeader
        shareContent={shareContent}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        onUploadFiles={handleUploadFiles}
      />

      <div className="mx-auto w-full max-w-6xl flex-1 overflow-y-auto px-6 py-6">
        {shareContent.allow_upload &&
        !(
          shareContent.max_uploads !== null &&
          shareContent.current_uploads >= shareContent.max_uploads
        ) ? (
          <ShareUploadZone
            shareId={shareId}
            token={token}
            maxUploadSize={shareContent.max_upload_size}
            folderId={
              shareContent.type === "bucket" ? currentFolderId : undefined
            }
            onReady={(fn) => {
              uploadFilesRef.current = fn;
            }}
          >
            {contentArea}
          </ShareUploadZone>
        ) : (
          contentArea
        )}
      </div>
    </div>
  );
};
