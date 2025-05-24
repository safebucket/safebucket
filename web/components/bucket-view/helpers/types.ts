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

export interface IShareWith {
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
  createBucket: (name: string, shareWith: IShareWith[]) => void;
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
}

export type IDownloadFileResponse = {
  url: string;
};
