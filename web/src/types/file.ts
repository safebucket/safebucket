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
  extension: string;
  folder_id?: string;
  status: FileStatus | null;
  created_at: string;
    deleted_at?: string;
  deleted_by?: string;
    original_path?: string;
}
