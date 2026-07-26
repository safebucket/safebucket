export interface FileWithPath {
  file: File;
  relativePath: string;
}

export interface StagedFile {
  id: string;
  file: File;
  relativePath: string;
  extension: string;
}

export interface IFilePartURL {
  part_number: number;
  url: string;
  size: number;
  headers?: Record<string, string>;
}

export type IUploadPresign =
  | {
      id: string;
      method: "post";
      url: string;
      body: Array<Record<string, string>>;
    }
  | { id: string; method: "put"; parts: Array<IFilePartURL> };

export interface IUploadContext {
  uploads: Array<IUpload>;
  startUpload: (
    files: Array<File>,
    bucketId: string,
    folderId: string | undefined,
    expiresAt: string | null,
  ) => void;
  cancelUpload: (uploadId: string) => void;
  clearUploads: () => void;
  hasActiveUploads: boolean;
}

export type UploadStatus = "uploading" | "success" | "error";

export interface IUpload {
  id: string;
  name: string;
  path: string;
  progress: number;
  status: UploadStatus;
  error?: Error;
}

export type FileSystemEntry = {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
};

export type FileSystemFileEntry = FileSystemEntry & {
  file: (callback: (file: File) => void) => void;
};

export type FileSystemDirectoryEntry = FileSystemEntry & {
  createReader: () => FileSystemDirectoryReader;
};

export type FileSystemDirectoryReader = {
  readEntries: (callback: (entries: Array<FileSystemEntry>) => void) => void;
};
