import type { UseMutationResult } from "@tanstack/react-query";
import type { IBucket } from "@/types/bucket.ts";

export interface IBucketsData {
  buckets: Array<IBucket>;
  isLoading: boolean;
  createBucketMutation: UseMutationResult<IBucket, Error, any>;
}

export type IDownloadFileResponse = {
  url: string;
};

export interface INotificationPreferences {
  upload_notifications: boolean;
  download_notifications: boolean;
}

export type IBucketMember = INotificationPreferences & {
  user_id?: string;
  email: string;
  first_name?: string;
  last_name?: string;
  group: string;
  status: "active" | "invited";
};
