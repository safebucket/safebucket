export interface IUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface IBucket {
  id: string;
  name: string;
}

export interface IFile {
  id: string;
  name: string;
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
  BUCKET_MEMBER_CREATED = "BUCKET_MEMBER_CREATED",
  BUCKET_MEMBER_UPDATED = "BUCKET_MEMBER_UPDATED",
  BUCKET_MEMBER_DELETED = "BUCKET_MEMBER_DELETED",
}

export interface IListBucketActivity {
  data: Array<IActivity>;
}

export interface MessageMapping {
  message: string;
  icon: any;
  iconColor: string;
  iconBg: string;
}

export interface IActivityData {
  activity: Array<IActivity>;
  error: string;
  isLoading: boolean;
}
