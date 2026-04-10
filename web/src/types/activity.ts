import type { IBucket } from "@/types/bucket.ts";
import type { IFile } from "@/types/file.ts";
import type { IFolder } from "@/types/folder.ts";

export interface IUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface IMFADevice {
  id: string;
  name: string;
}

export interface IActivity {
  domain: string;
  user_id?: string;
  user?: IUser;
  action: string;
  object_type: string;
  bucket_id?: string;
  bucket?: IBucket;
  file_id?: string;
  file?: IFile;
  folder_id?: string;
  folder?: IFolder;
  share_id?: string;
  share?: { id: string; name: string };
  timestamp: string;
  message: ActivityMessage;
  bucket_member_email?: string;
  mfa_device?: IMFADevice;
}

export enum ActivityMessage {
  BUCKET_CREATED = "BUCKET_CREATED",
  BUCKET_DELETED = "BUCKET_DELETED",
  FILE_UPLOADED = "FILE_UPLOADED",
  FILE_DOWNLOADED = "FILE_DOWNLOADED",
  FILE_UPDATED = "FILE_UPDATED",
  FILE_DELETED = "FILE_DELETED",
  FILE_EXPIRED = "FILE_EXPIRED",
  FILE_TRASHED = "FILE_TRASHED",
  FILE_RESTORED = "FILE_RESTORED",
  FOLDER_CREATED = "FOLDER_CREATED",
  FOLDER_UPDATED = "FOLDER_UPDATED",
  FOLDER_TRASHED = "FOLDER_TRASHED",
  FOLDER_DELETED = "FOLDER_DELETED",
  FOLDER_RESTORED = "FOLDER_RESTORED",
  BUCKET_MEMBER_CREATED = "BUCKET_MEMBER_CREATED",
  BUCKET_MEMBER_UPDATED = "BUCKET_MEMBER_UPDATED",
  BUCKET_MEMBER_DELETED = "BUCKET_MEMBER_DELETED",
  MFA_DEVICE_ENROLLED = "MFA_DEVICE_ENROLLED",
  MFA_DEVICE_VERIFIED = "MFA_DEVICE_VERIFIED",
  MFA_DEVICE_UPDATED = "MFA_DEVICE_UPDATED",
  MFA_DEVICE_REMOVED = "MFA_DEVICE_REMOVED",
  SHARE_FILE_DOWNLOADED = "SHARE_FILE_DOWNLOADED",
  SHARE_FILE_UPLOADED = "SHARE_FILE_UPLOADED",
}

export interface IListBucketActivity {
  data: Array<IActivity>;
}
