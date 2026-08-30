import { useCallback, useMemo, useState } from "react";

import { format } from "date-fns";
import i18n from "i18next";

import { toast } from "sonner";

import type { RowSelectionState } from "@tanstack/react-table";
import type { IBucket } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import type { ZipDownloadEntry } from "@/lib/zip-download";
import { api_downloadFile } from "@/components/file-actions/helpers/api";
import { fetchDownloadBlob, zipDownload } from "@/lib/zip-download";
import { errorToast } from "@/lib/toast";
import { FileStatus } from "@/types/file.ts";

const MAX_TOTAL_BYTES = 1024 * 1024 * 1024;
const MAX_FILE_COUNT = 100;
const PROGRESS_TOAST_ID = "bulk-download-progress";

interface IBlockedInfo {
  count: number;
  bytes: number;
}

interface IUseBulkDownloadArgs {
  bucket: IBucket;
  rowSelection: RowSelectionState;
  clearRowSelection: () => void;
}

const isDownloadableFile = (file: IFile): boolean =>
  file.status === FileStatus.uploaded && !file.deleted_at;

export const collectFilesForSelection = (
  bucket: IBucket,
  selection: RowSelectionState,
): Array<ZipDownloadEntry> => {
  const selectedIds = Object.keys(selection).filter((id) => selection[id]);
  if (selectedIds.length === 0) {
    return [];
  }

  const fileById = new Map<string, IFile>();
  const folderById = new Map<string, IFolder>();
  const childFoldersByParent = new Map<string, Array<IFolder>>();
  const childFilesByParent = new Map<string, Array<IFile>>();
  for (const folder of bucket.folders) {
    folderById.set(folder.id, folder);
    if (folder.folder_id) {
      const list = childFoldersByParent.get(folder.folder_id) ?? [];
      list.push(folder);
      childFoldersByParent.set(folder.folder_id, list);
    }
  }
  for (const file of bucket.files) {
    fileById.set(file.id, file);
    if (file.folder_id) {
      const list = childFilesByParent.get(file.folder_id) ?? [];
      list.push(file);
      childFilesByParent.set(file.folder_id, list);
    }
  }

  const entries: Array<ZipDownloadEntry> = [];
  const seen = new Set<string>();

  const walkFolder = (folder: IFolder, basePath: string) => {
    const folderPath = `${basePath}${folder.name}/`;
    const childFiles = childFilesByParent.get(folder.id) ?? [];
    for (const file of childFiles) {
      if (!isDownloadableFile(file) || seen.has(file.id)) continue;
      seen.add(file.id);
      entries.push({ file, zipPath: `${folderPath}${file.name}` });
    }
    const childFolders = childFoldersByParent.get(folder.id) ?? [];
    for (const child of childFolders) {
      walkFolder(child, folderPath);
    }
  };

  for (const id of selectedIds) {
    const file = fileById.get(id);
    if (file) {
      if (!isDownloadableFile(file) || seen.has(file.id)) continue;
      seen.add(file.id);
      entries.push({ file, zipPath: file.name });
      continue;
    }
    const folder = folderById.get(id);
    if (folder) {
      walkFolder(folder, "");
    }
  }

  return entries;
};

export const useBulkDownload = ({
  bucket,
  rowSelection,
  clearRowSelection,
}: IUseBulkDownloadArgs) => {
  const [blocked, setBlocked] = useState<IBlockedInfo | null>(null);
  const [isRunning, setIsRunning] = useState(false);

  const entries = useMemo(
    () => collectFilesForSelection(bucket, rowSelection),
    [bucket, rowSelection],
  );

  const totalBytes = useMemo(
    () => entries.reduce((sum, e) => sum + e.file.size, 0),
    [entries],
  );

  const fetchFile = useCallback(
    async (file: IFile) => {
      const response = await api_downloadFile(bucket.id, file.id);
      return fetchDownloadBlob(response.url);
    },
    [bucket.id],
  );

  const run = useCallback(async () => {
    const progressTitle = i18n.t("bucket.bulk_download.progress_title");
    setIsRunning(true);
    toast(progressTitle, {
      id: PROGRESS_TOAST_ID,
      description: i18n.t("bucket.bulk_download.progress", {
        done: 0,
        total: entries.length,
      }),
      duration: Infinity,
    });

    try {
      const date = format(new Date(), "yyyyMMdd-HHmmss");
      const safeBucketName = bucket.name.replace(/[^a-zA-Z0-9-_]/g, "_");
      const result = await zipDownload({
        entries,
        archiveName: `${safeBucketName}-${date}.zip`,
        fetchFile,
        onProgress: (completed, total) => {
          toast(progressTitle, {
            id: PROGRESS_TOAST_ID,
            description: i18n.t("bucket.bulk_download.progress", {
              done: completed,
              total,
            }),
            duration: Infinity,
          });
        },
      });

      if (result.failures.length === result.total) {
        errorToast(new Error("bulk_download_all_failed"));
        return;
      }

      if (result.failures.length > 0) {
        errorToast(
          i18n.t("bucket.bulk_download.partial_failure_title"),
          i18n.t("bucket.bulk_download.partial_failure", {
            count: result.failures.length,
            files: result.failures.slice(0, 5).join(", "),
          }),
        );
      } else {
        toast.success(
          i18n.t("bucket.bulk_download.success", { count: entries.length }),
        );
      }
      clearRowSelection();
    } catch (error) {
      errorToast(error as Error);
    } finally {
      toast.dismiss(PROGRESS_TOAST_ID);
      setIsRunning(false);
    }
  }, [bucket.name, clearRowSelection, entries, fetchFile]);

  const start = useCallback(() => {
    if (entries.length === 0 || isRunning) return;
    if (totalBytes > MAX_TOTAL_BYTES || entries.length > MAX_FILE_COUNT) {
      setBlocked({ count: entries.length, bytes: totalBytes });
      return;
    }
    void run();
  }, [entries.length, isRunning, run, totalBytes]);

  const dismissBlocked = useCallback(() => setBlocked(null), []);

  return {
    start,
    dismissBlocked,
    blocked,
    isRunning,
    fileCount: entries.length,
    totalBytes,
    maxBytes: MAX_TOTAL_BYTES,
    maxFiles: MAX_FILE_COUNT,
  };
};
