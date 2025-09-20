export enum FileStatus {
  uploading = "uploading",
  uploaded = "uploaded",
  deletion_scheduled = "deletion_scheduled",
}

export interface File {
  id: string;
  name: string;
  size: number;
  type: FileType;
  extension: string;
  path: string;
  files: Array<File>;
  status: FileStatus | null;
  created_at: string;
}

export enum FileType {
  file = "file",
  folder = "folder",
}

export interface Invites {
  email: string;
  group: string;
}

export interface Bucket {
  id: string;
  name: string;
  files: Array<File>;
  created_by: string;
  created_at: string;
  updated_at: string;
}
