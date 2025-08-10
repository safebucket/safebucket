import { api } from "@/lib/api";

import { IInvites } from "@/components/bucket-view/helpers/types";

export const api_updateMembers = (bucketId: string, members: IInvites[]) =>
  api.put(`/buckets/${bucketId}/members`, { members });
