import type { UseMutationResult } from "@tanstack/react-query";
import type { IBucket } from "@/types/bucket.ts";

export interface IMembers {
  email: string;
  group: string;
}

export interface IBucketsData {
  buckets: Array<IBucket>;
  isLoading: boolean;
  createBucketMutation: UseMutationResult<IBucket, Error, any>;
  isDialogOpen: boolean;
  setIsDialogOpen: (isOpen: boolean) => void;
}

export enum BucketViewMode {
  List = "list",
  Grid = "grid",
  Activity = "activity",
  Trash = "trash",
  Settings = "settings",
}

export type IDownloadFileResponse = {
  url: string;
};

export interface IBucketMember {
  user_id?: string;
  email: string;
  first_name?: string;
  last_name?: string;
  group: string;
  status: "active" | "invited";
}
