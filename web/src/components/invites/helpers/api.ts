import type {
  IChallengeValidationResponse,
  ICreateChallengeResponse,
} from "@/components/invites/helpers/types";
import { api } from "@/lib/api";

export const api_createChallenge = (invitationId: string, email: string) =>
  api.post<ICreateChallengeResponse>(`/invites/${invitationId}/challenges`, {
    email,
  });

export const api_validateChallenge = (
  invitationId: string,
  challengeId: string,
  data: { code: string; new_password: string },
) =>
  api.post<IChallengeValidationResponse>(
    `/invites/${invitationId}/challenges/${challengeId}/validate`,
    data,
  );
