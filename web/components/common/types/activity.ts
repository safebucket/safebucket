import { IUser } from "@/components/auth-view/types/session";
import { IBucket, IFile } from "@/components/bucket-view/helpers/types";

export type IListBucketActivity = {
  data: IActivity[];
};

export interface IActivityData {
  activity: IActivity[];
  error: string;
  isLoading: boolean;
}

export interface IActivity {
  domain: string;
  user_id: string;
  user: IUser;
  action: string;
  object_type: string;
  bucket_id?: string;
  bucket?: IBucket;
  file_id?: string;
  file?: IFile;
  timestamp: string;
  message: ActivityMessage;
}

export enum ActivityMessage {
  BUCKET_CREATED = "BUCKET_CREATED",
  FILE_UPLOADED = "FILE_UPLOADED",
  FILE_DOWNLOADED = "FILE_DOWNLOADED",
  FILE_UPDATED = "FILE_UPDATED",
  FILE_DELETED = "FILE_DELETED",
}
