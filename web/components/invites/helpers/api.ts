import { api } from "@/lib/api";

export const api_createChallenge = (invitationId: string, email: string) =>
  api.post(`/invites/${invitationId}/challenges`, {
    email,
  });
