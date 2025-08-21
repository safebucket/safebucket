export interface File {
  id: string;
  name: string;
  size: number;
  type: FileType;
  extension: string;
  path: string;
  files: File[];
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
  files: File[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface BucketsResponse {
  data: Bucket[];
}
