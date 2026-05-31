import { useCallback, useMemo, useState } from "react";

import { format } from "date-fns";
import i18n from "i18next";
import JSZip from "jszip";

import type { RowSelectionState } from "@tanstack/react-table";
import type { IBucket } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import { api_downloadFile } from "@/components/file-actions/helpers/api";
import {
  errorToast,
  successToast,
  toast,
} from "@/components/ui/hooks/use-toast";
import { triggerBlobDownload } from "@/lib/download";
import { FileStatus } from "@/types/file.ts";

const MAX_TOTAL_BYTES = 1024 * 1024 * 1024;
const MAX_FILE_COUNT = 100;
const CONCURRENCY = 3;

interface IBulkDownloadEntry {
  file: IFile;
  zipPath: string;
}

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
): Array<IBulkDownloadEntry> => {
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

  const entries: Array<IBulkDownloadEntry> = [];
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

const fetchAsBlob = async (url: string): Promise<Blob> => {
  const res = await fetch(url);
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return res.blob();
};

const runWithConcurrency = async <T>(
  items: Array<T>,
  limit: number,
  worker: (item: T) => Promise<void>,
): Promise<void> => {
  const queue = items.slice();
  const runners = Array.from(
    { length: Math.min(limit, items.length) },
    async () => {
      while (queue.length > 0) {
        await worker(queue.shift()!);
      }
    },
  );
  await Promise.all(runners);
};

export const useBulkDownload = ({
  bucket,
  rowSelection,
  clearRowSelection,
}: IUseBulkDownloadArgs) => {
  const [isRunning, setIsRunning] = useState(false);
  const [blocked, setBlocked] = useState<IBlockedInfo | null>(null);

  const entries = useMemo(
    () => collectFilesForSelection(bucket, rowSelection),
    [bucket, rowSelection],
  );

  const totalBytes = useMemo(
    () => entries.reduce((sum, e) => sum + e.file.size, 0),
    [entries],
  );

  const run = useCallback(async () => {
    if (entries.length === 0) {
      return;
    }
    setIsRunning(true);

    const zip = new JSZip();
    let done = 0;
    const failures: Array<string> = [];

    const handle = toast({
      title: i18n.t("bucket.bulk_download.progress_title"),
      description: i18n.t("bucket.bulk_download.progress", {
        done: 0,
        total: entries.length,
      }),
    });

    try {
      await runWithConcurrency(entries, CONCURRENCY, async (entry) => {
        try {
          const res = await api_downloadFile(bucket.id, entry.file.id);
          const blob = await fetchAsBlob(res.url);
          zip.file(entry.zipPath, blob);
        } catch (err) {
          failures.push(entry.file.name);
        } finally {
          done += 1;
          handle.update({
            id: handle.id,
            title: i18n.t("bucket.bulk_download.progress_title"),
            description: i18n.t("bucket.bulk_download.progress", {
              done,
              total: entries.length,
            }),
          } as any);
        }
      });

      if (failures.length === entries.length) {
        handle.dismiss();
        errorToast(new Error("bulk_download_all_failed"));
        return;
      }

      const archive = await zip.generateAsync({ type: "blob" });
      const stamp = format(new Date(), "yyyyMMdd-HHmmss");
      const safeBucketName = bucket.name.replace(/[^a-zA-Z0-9-_]/g, "_");
      triggerBlobDownload(archive, `${safeBucketName}-${stamp}.zip`);

      handle.dismiss();
      if (failures.length > 0) {
        toast({
          variant: "destructive",
          title: i18n.t("bucket.bulk_download.partial_failure_title"),
          description: i18n.t("bucket.bulk_download.partial_failure", {
            count: failures.length,
            files: failures.slice(0, 5).join(", "),
          }),
        });
      } else {
        successToast(
          i18n.t("bucket.bulk_download.success", { count: entries.length }),
        );
      }
      clearRowSelection();
    } catch (err) {
      handle.dismiss();
      errorToast(err as Error);
    } finally {
      setIsRunning(false);
    }
  }, [bucket.id, bucket.name, entries, clearRowSelection]);

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
