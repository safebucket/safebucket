export interface ICreateFile {
  id: string;
  path: string;
  url: string;
}

export interface IStartUploadData {
  files: File[];
}

export interface IUploadContext {
  uploads: IUpload[];

  startUpload(data: IStartUploadData, path: string, bucketId?: string): void;
}

export enum UploadStatus {
  uploading = "uploading",
  success = "success",
  failed = "failed",
}

export interface IUpload {
  id: string;
  name: string;
  progress: number;
  status: UploadStatus;
}
