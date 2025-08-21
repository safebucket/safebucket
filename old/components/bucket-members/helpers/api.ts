import { api } from "@/lib/api";

import { IMembers } from "@/components/bucket-view/helpers/types";

export const api_updateMembers = (bucketId: string, members: IMembers[]) =>
  api.put(`/buckets/${bucketId}/members`, { members });
