export interface ISharingOptions {
  id: string;
  file_id: string;
  expires_at?: string;
  max_downloads?: number;
  download_count: number;
  created_at: string;
  updated_at: string;
}

export interface ISharingOptionsInput {
  expires_at?: string;
  max_downloads?: number;
}
