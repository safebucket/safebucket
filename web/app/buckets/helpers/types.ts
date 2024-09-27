import {
  SubmitHandler,
  UseFormHandleSubmit,
  UseFormRegister,
} from "react-hook-form";

export interface IFile {
  id: number;
  name: string;
  size: string;
  modified: string;
  type: string;
  selected: boolean;
}

export interface Bucket {
  id: string;
  name: string;
  files: IFile[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface IBucketsData {
  buckets: Bucket[];
  error: string;
  isLoading: boolean;
  createBucket: SubmitHandler<IBucketForm>;
  register: UseFormRegister<IBucketForm>;
  handleSubmit: UseFormHandleSubmit<IBucketForm>;
  isDialogOpen: boolean;
  setIsDialogOpen: (isOpen: boolean) => void;
}

export interface IBucketData {
  bucket: Bucket | undefined;
  error: string;
  isLoading: boolean;
}

export type IBucketForm = {
  name: string;
};
