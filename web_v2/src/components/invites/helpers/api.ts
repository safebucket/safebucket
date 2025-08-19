import { api } from "@/lib/api";

import type {
  IChallengeValidationResponse,
  ICreateChallengeResponse,
} from "@/components/invites/helpers/types";

export const api_createChallenge = (invitationId: string, email: string) =>
  api.post<ICreateChallengeResponse>(`/invites/${invitationId}/challenges`, {
    email,
  });

export const api_validateChallenge = (
  invitationId: string,
  challengeId: string,
  code: string,
) =>
  api.post<IChallengeValidationResponse>(
    `/invites/${invitationId}/challenges/${challengeId}/validate`,
    {
      code,
    },
  );
