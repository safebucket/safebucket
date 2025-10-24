import type { IBucket } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";

export interface IUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
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
  bucket_member_email?: string;
}

export enum ActivityMessage {
  BUCKET_CREATED = "BUCKET_CREATED",
  FILE_UPLOADED = "FILE_UPLOADED",
  FILE_DOWNLOADED = "FILE_DOWNLOADED",
  FILE_UPDATED = "FILE_UPDATED",
  FILE_DELETED = "FILE_DELETED",
  FILE_TRASHED = "FILE_TRASHED",
  FILE_RESTORED = "FILE_RESTORED",
  FILE_PURGED = "FILE_PURGED",
  FOLDER_TRASHED = "FOLDER_TRASHED",
  FOLDER_RESTORED = "FOLDER_RESTORED",
  FOLDER_PURGED = "FOLDER_PURGED",
  BUCKET_MEMBER_CREATED = "BUCKET_MEMBER_CREATED",
  BUCKET_MEMBER_UPDATED = "BUCKET_MEMBER_UPDATED",
  BUCKET_MEMBER_DELETED = "BUCKET_MEMBER_DELETED",
}

export interface IListBucketActivity {
  data: Array<IActivity>;
}
