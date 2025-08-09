import { api } from "@/lib/api";

import {
  IInviteResponse,
  IInvites,
} from "@/components/bucket-view/helpers/types";

export const api_updateMembers = (bucketId: string, invites: IInvites[]) =>
  api.put<IInviteResponse[]>(`/buckets/${bucketId}/members`, { invites });
