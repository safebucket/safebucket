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
  password?: string;
  allow_upload: boolean;
  max_uploads?: number;
  max_upload_size?: number;
}

export interface IPublicShareResponse {
  id: string;
  name: string;
  type: ShareScope;
  allow_upload: boolean;
  max_upload_size: number | null;
  max_uploads: number | null;
  current_uploads: number;
  expires_at: string | null;
  max_views: number | null;
  current_views: number;
  files: Array<IFile>;
  folders: Array<IFolder>;
}

export interface IShareAuthResponse {
  token: string;
}

export interface IShareUploadBody {
  name: string;
  size: number;
  folder_id?: string;
}

export interface IFileTransferResponse {
  id: string;
  url: string;
  body?: Record<string, string>;
}
