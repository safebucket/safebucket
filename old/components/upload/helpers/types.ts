interface ICreateFileBody {
  bucket: string;
  key: string;
  policy: string;
  "x-amz-algorithm": string;
  "x-amz-credential": string;
  "x-amz-date": string;
  "x-amz-signature": string;
}

export interface ICreateFile {
  id: string;
  path: string;
  url: string;
  body: ICreateFileBody;
}

export interface IUploadContext {
  uploads: IUpload[];
  startUpload: (files: FileList, path: string, bucketId: string) => void;
}

export enum UploadStatus {
  uploading = "uploading",
  success = "success",
  failed = "failed",
}

export interface IUpload {
  id: string;
  name: string;
  path: string;
  progress: number;
  status: UploadStatus;
}

// File System API type definitions for drag & drop functionality
export type FileSystemEntry = {
  isFile: boolean;
  isDirectory: boolean;
  name: string;
};

export type FileSystemFileEntry = FileSystemEntry & {
  file(callback: (file: File) => void): void;
};

export type FileSystemDirectoryEntry = FileSystemEntry & {
  createReader(): FileSystemDirectoryReader;
};

export type FileSystemDirectoryReader = {
  readEntries(callback: (entries: FileSystemEntry[]) => void): void;
};
