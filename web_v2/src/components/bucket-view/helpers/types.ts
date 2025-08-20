export interface IFile {
  id: string;
  name: string;
  size: number;
  type: FileType;
  extension: string;
  path: string;
  files: Array<IFile>;
  created_at: string;
}

export enum FileType {
  file = "file",
  folder = "folder",
}

export interface IMembers {
  email: string;
  group: string;
}

export interface IBucket {
  id: string;
  name: string;
  files: Array<IFile>;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IBucketsData {
  buckets: Array<IBucket>;
  error: string;
  isLoading: boolean;
  createBucketAndInvites: (name: string, shareWith: Array<IMembers>) => void;
  isDialogOpen: boolean;
  setIsDialogOpen: (isOpen: boolean) => void;
}

export type IListBuckets = {
  data: Array<IBucket>;
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
  email: string;
  group: string;
  status: string;
};

export interface IBucketMember {
  user_id?: string;
  email: string;
  first_name?: string;
  last_name?: string;
  group: string;
  status: "active" | "invited";
}
