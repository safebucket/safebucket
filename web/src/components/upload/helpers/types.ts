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
  uploads: Array<IUpload>;
  startUpload: (
    files: FileList,
    bucketId: string,
    folderId: string | undefined,
    settings?: IUploadSettings,
  ) => void;
  cancelUpload: (uploadId: string) => void;
  hasActiveUploads: boolean;
}

export type UploadStatus = "uploading" | "success" | "error";

export interface IUploadSettings {
  expiresAt?: Date;
  maxDownloads?: number;
}

export interface IUpload {
  id: string;
  name: string;
  path: string;
  progress: number;
  status: UploadStatus;
  error?: Error;
  settings?: IUploadSettings;
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
