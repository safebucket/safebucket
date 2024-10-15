export interface ICreateFile {
  id: string;
  url: string;
}

export interface IStartUploadData {
  files: File[];
}

export interface IFileTransferContext {
  transfers: IUpload[];

  startUpload(data: IStartUploadData, bucketId?: string): void;
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
