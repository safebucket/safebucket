export enum FileStatus {
  uploading = "uploading",
  uploaded = "uploaded",
  deleting = "deleting",
  deleted = "deleted",
  restoring = "restoring",
}

export interface IFile {
  id: string;
  name: string;
  size: number;
  extension: string;
  folder_id?: string;
  status: FileStatus | null;
  created_at: string;
  deleted_at: string | null;
  deleted_by?: string;
  original_path?: string;
}
