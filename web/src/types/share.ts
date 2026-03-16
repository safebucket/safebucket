export interface IShare {
  id: string;
  name: string;
  bucket_id: string;
  folder_id: string | null;
  type: "files" | "folder" | "bucket";
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
