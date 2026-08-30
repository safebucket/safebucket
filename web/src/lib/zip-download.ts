import JSZip from "jszip";

import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";
import { triggerBlobDownload } from "@/lib/download.ts";

const DEFAULT_CONCURRENCY = 3;

export interface ZipDownloadEntry {
  file: IFile;
  zipPath: string;
}

interface IZipDownloadOptions {
  entries: Array<ZipDownloadEntry>;
  archiveName: string;
  fetchFile: (file: IFile) => Promise<Blob>;
  onProgress?: (completed: number, total: number) => void;
  concurrency?: number;
}

export interface ZipDownloadResult {
  failures: Array<string>;
  total: number;
}

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

export const fetchDownloadBlob = async (url: string): Promise<Blob> => {
  const response = await fetch(url);
  if (!response.ok) throw new Error(`HTTP ${response.status}`);
  return response.blob();
};

export const createZipEntries = (
  files: Array<IFile>,
  folders: Array<IFolder>,
): Array<ZipDownloadEntry> => {
  const foldersById = new Map(folders.map((folder) => [folder.id, folder]));

  return files.map((file) => {
    const path: Array<string> = [];
    const seen = new Set<string>();
    let folderId = file.folder_id;

    while (folderId && !seen.has(folderId)) {
      seen.add(folderId);
      const folder = foldersById.get(folderId);
      if (!folder) break;
      path.unshift(folder.name);
      folderId = folder.folder_id;
    }

    return { file, zipPath: [...path, file.name].join("/") };
  });
};

export const zipDownload = async ({
  entries,
  archiveName,
  fetchFile,
  onProgress,
  concurrency = DEFAULT_CONCURRENCY,
}: IZipDownloadOptions): Promise<ZipDownloadResult> => {
  const zip = new JSZip();
  const failures: Array<string> = [];
  let completed = 0;

  await runWithConcurrency(entries, concurrency, async (entry) => {
    try {
      zip.file(entry.zipPath, await fetchFile(entry.file));
    } catch {
      failures.push(entry.file.name);
    } finally {
      completed += 1;
      onProgress?.(completed, entries.length);
    }
  });

  if (failures.length < entries.length) {
    const archive = await zip.generateAsync({ type: "blob" });
    triggerBlobDownload(archive, archiveName);
  }

  return { failures, total: entries.length };
};
