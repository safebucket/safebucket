import type { IMembers } from "@/components/bucket-view/helpers/types";
import { api } from "@/lib/api";

export const api_updateMembers = (bucketId: string, members: Array<IMembers>) =>
  api.put(`/buckets/${bucketId}/members`, { members });
