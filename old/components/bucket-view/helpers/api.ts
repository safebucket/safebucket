import { api } from "@/lib/api";

import { IBucket, IMembers } from "@/components/bucket-view/helpers/types";

export const api_createBucket = (name: string) =>
  api.post<IBucket>("/buckets", { name });

export const api_addMembers = (bucketId: string, members: IMembers[]) =>
  api.put(`/buckets/${bucketId}/members`, { members });
