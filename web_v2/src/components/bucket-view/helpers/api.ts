import { api } from "@/lib/api";

import type {
  IBucket,
  IInviteResponse,
  IMembers,
} from "@/components/bucket-view/helpers/types";

export const api_createBucket = (name: string) =>
  api.post<IBucket>("/buckets", { name });

export const api_createInvites = (bucket_id: string, invites: IMembers[]) =>
  api.post<IInviteResponse[]>("/invites", { bucket_id, invites });
