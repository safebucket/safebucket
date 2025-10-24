export enum FileType {
  file = "file",
  folder = "folder",
}

export enum FileStatus {
  uploading = "uploading",
  uploaded = "uploaded",
  deleting = "deleting",
  trashed = "trashed",
  restoring = "restoring",
}

export interface IUser {
  id: string;
  email: string;
  first_name?: string;
  last_name?: string;
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
  trashed_at?: string;
  trashed_by?: string;
  trashed_user?: IUser;
}
