export interface IFile {
  id: string;
  name: string;
  size: number;
  type: FileType;
  extension: string;
  path: string;
  files: IFile[];
  created_at: string;
}

export enum FileType {
  file = "file",
  folder = "folder",
}

export interface IInvites {
  email: string;
  group: string;
}

export interface IBucket {
  id: string;
  name: string;
  files: IFile[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IBucketsData {
  buckets: IBucket[];
  error: string;
  isLoading: boolean;
  createBucketAndInvites: (name: string, shareWith: IInvites[]) => void;
  isDialogOpen: boolean;
  setIsDialogOpen: (isOpen: boolean) => void;
}

export interface IBucketData {
  bucket: IBucket | undefined;
  error: string;
  isLoading: boolean;
}

export type IListBuckets = {
  data: IBucket[];
};

export enum BucketViewMode {
  List = "list",
  Grid = "grid",
  Activity = "activity",
  Settings = "settings",
}

export type IDownloadFileResponse = {
  url: string;
};

export type IInviteResponse = {
  email: string,
  group: string
  status: string
};

export interface IBucketMember {
  user_id?: string;
  email: string;
  first_name?: string;
  last_name?: string;
  role: string;
  status: "active" | "invited";
}
