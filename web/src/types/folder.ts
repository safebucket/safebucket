export enum FolderStatus {
  created = "created",
  deleted = "deleted",
  restoring = "restoring",
}

export interface IFolder {
  id: string;
  name: string;
  folder_id?: string;
  bucket_id: string;
  status: FolderStatus;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  deleted_by?: string;
  original_path?: string;
}
