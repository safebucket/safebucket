export enum FileType {
  file = "file",
  folder = "folder",
}

export enum FileStatus {
  uploading = "uploading",
  uploaded = "uploaded",
  deleting = "deleting",
}

export interface IFile {
  id: string;
  name: string;
  size: number;
  type: FileType;
  extension: string;
  path: string;
  files: Array<IFile>;
  status: FileStatus | null;
  created_at: string;
}
