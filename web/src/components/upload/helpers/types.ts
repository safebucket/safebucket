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
    folderId: string | null,
  ) => void;
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
