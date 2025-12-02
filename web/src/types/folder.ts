import type { FileStatus } from "./file";

export interface IFolder {
  id: string;
  name: string;
  folder_id?: string;
  bucket_id: string;
  status: FileStatus | null;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
  deleted_by?: string;
}
