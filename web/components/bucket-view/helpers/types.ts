import {
  SubmitHandler,
  UseFormHandleSubmit,
  UseFormRegister,
} from "react-hook-form";

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
  createBucket: SubmitHandler<IBucketForm>;
  register: UseFormRegister<IBucketForm>;
  handleSubmit: UseFormHandleSubmit<IBucketForm>;
  isDialogOpen: boolean;
  setIsDialogOpen: (isOpen: boolean) => void;
}

export interface IBucketData {
  bucket: IBucket | undefined;
  error: string;
  isLoading: boolean;
}

export type IBucketForm = {
  name: string;
};

export type IListBuckets = {
  data: IBucket[];
};

export enum BucketViewMode {
  List = "list",
  Grid = "grid",
}

export type IDownloadFileResponse = {
  url: string;
};
