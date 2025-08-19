export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
}

export interface Bucket {
  id: string;
  name: string;
}

export interface File {
  id: string;
  name: string;
}

export interface Activity {
  domain: string;
  user_id: string;
  user: User;
  action: string;
  object_type: string;
  bucket_id?: string;
  bucket?: Bucket;
  file_id?: string;
  file?: File;
  timestamp: string;
  message: ActivityMessage;
  invited_email?: string;
}

export enum ActivityMessage {
  BUCKET_CREATED = "BUCKET_CREATED",
  FILE_UPLOADED = "FILE_UPLOADED",
  FILE_DOWNLOADED = "FILE_DOWNLOADED",
  FILE_UPDATED = "FILE_UPDATED",
  FILE_DELETED = "FILE_DELETED",
  USER_INVITED = "USER_INVITED",
}

export interface ActivityResponse {
  data: Activity[];
}

export interface MessageMapping {
  message: string;
  icon: any;
  iconColor: string;
  iconBg: string;
}
