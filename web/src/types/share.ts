import type { IFile } from "@/types/file";
import type { IFolder } from "@/types/folder";

export type ShareScope = "files" | "folder" | "bucket";

export interface IShare {
  id: string;
  name: string;
  bucket_id: string;
  folder_id: string | null;
  type: ShareScope;
  expires_at: string | null;
  max_views: number | null;
  current_views: number;
  password_protected: boolean;
  allow_upload: boolean;
  max_uploads: number | null;
  current_uploads: number;
  max_upload_size: number | null;
  files: Array<IShareFile> | null;
  created_by: string;
  created_at: string;
}

export interface IShareFile {
  id: string;
  share_id: string;
  file_id: string;
}

export interface IShareCreateBody {
  name: string;
  type: ShareScope;
  file_ids?: Array<string>;
  folder_id?: string | null;
  expires_at?: string;
  max_views?: number;
  allow_upload: boolean;
  max_uploads?: number;
  max_upload_size?: number;
}

export interface IPublicShareContent {
  files: Array<IFile>;
  folders: Array<IFolder>;
}

export type IConsumeShareResponse =
  | { password_required: true }
  | { password_required: false; share: IShare; content: IPublicShareContent };
