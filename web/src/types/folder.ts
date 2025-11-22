import type { FileStatus, IUser } from "./file";

export interface IFolder {
  id: string;
  name: string;
  folder_id?: string;
  bucket_id: string;
  status: FileStatus | null;
  created_at: string;
  updated_at: string;
  trashed_at?: string;
  trashed_by?: string;
  trashed_user?: IUser;
}
